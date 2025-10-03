package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

// Event structures
type Event struct {
	ID        string      `json:"id"`
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Payload   interface{} `json:"payload"`
}

type MovieEvent struct {
	MovieID     int      `json:"movie_id"`
	Title       string   `json:"title"`
	Action      string   `json:"action"`
	UserID      int      `json:"user_id,omitempty"`
	Rating      float64  `json:"rating,omitempty"`
	Genres      []string `json:"genres,omitempty"`
	Description string   `json:"description,omitempty"`
}

type UserEvent struct {
	UserID    int       `json:"user_id"`
	Username  string    `json:"username,omitempty"`
	Email     string    `json:"email,omitempty"`
	Action    string    `json:"action"`
	Timestamp time.Time `json:"timestamp"`
}

type PaymentEvent struct {
	PaymentID  int       `json:"payment_id"`
	UserID     int       `json:"user_id"`
	Amount     float64   `json:"amount"`
	Status     string    `json:"status"`
	Timestamp  time.Time `json:"timestamp"`
	MethodType string    `json:"method_type,omitempty"`
}

type EventResponse struct {
	Status    string `json:"status"`
	Partition int    `json:"partition"`
	Offset    int64  `json:"offset"`
	Event     Event  `json:"event"`
}

// EventService handles Kafka operations
type EventService struct {
	kafkaBrokers []string
	producers    map[string]*kafka.Writer
	consumers    map[string]*kafka.Reader
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
}

// NewEventService creates a new event service
func NewEventService(brokers []string) *EventService {
	ctx, cancel := context.WithCancel(context.Background())

	service := &EventService{
		kafkaBrokers: brokers,
		producers:    make(map[string]*kafka.Writer),
		consumers:    make(map[string]*kafka.Reader),
		ctx:          ctx,
		cancel:       cancel,
	}

	// Initialize producers for each topic
	topics := []string{"movie-events", "user-events", "payment-events"}
	for _, topic := range topics {
		service.producers[topic] = &kafka.Writer{
			Addr:     kafka.TCP(brokers...),
			Topic:    topic,
			Balancer: &kafka.LeastBytes{},
		}
	}

	// Start consumers
	service.startConsumers()

	return service
}

// startConsumers starts Kafka consumers for all topics
func (es *EventService) startConsumers() {
	topics := []string{"movie-events", "user-events", "payment-events"}

	for _, topic := range topics {
		reader := kafka.NewReader(kafka.ReaderConfig{
			Brokers:     es.kafkaBrokers,
			Topic:       topic,
			GroupID:     "events-service-group",
			StartOffset: kafka.LastOffset,
			MaxBytes:    10e6, // 10MB
		})

		es.consumers[topic] = reader

		// Start consumer goroutine
		es.wg.Add(1)
		go es.consumeMessages(topic, reader)
	}
}

// consumeMessages consumes messages from a Kafka topic
func (es *EventService) consumeMessages(topic string, reader *kafka.Reader) {
	defer es.wg.Done()

	log.Printf("Starting consumer for topic: %s", topic)

	for {
		select {
		case <-es.ctx.Done():
			log.Printf("Stopping consumer for topic: %s", topic)
			return
		default:
			msg, err := reader.ReadMessage(es.ctx)
			if err != nil {
				if err == context.Canceled {
					return
				}
				log.Printf("Error reading message from %s: %v", topic, err)
				continue
			}

			// Process the message
			log.Printf("[CONSUMER] Topic: %s, Partition: %d, Offset: %d, Key: %s, Value: %s",
				topic, msg.Partition, msg.Offset, string(msg.Key), string(msg.Value))

			// Parse and log the event
			var event Event
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				log.Printf("Error parsing event: %v", err)
				continue
			}

			log.Printf("[EVENT PROCESSED] Type: %s, ID: %s, Timestamp: %s, Payload: %+v",
				event.Type, event.ID, event.Timestamp.Format(time.RFC3339), event.Payload)
		}
	}
}

// PublishEvent publishes an event to Kafka
func (es *EventService) PublishEvent(topic string, event Event) (int, int64, error) {
	es.mu.RLock()
	producer, exists := es.producers[topic]
	es.mu.RUnlock()

	if !exists {
		return 0, 0, fmt.Errorf("producer for topic %s not found", topic)
	}

	eventBytes, err := json.Marshal(event)
	if err != nil {
		return 0, 0, err
	}

	msg := kafka.Message{
		Key:   []byte(event.ID),
		Value: eventBytes,
	}

	err = producer.WriteMessages(es.ctx, msg)
	if err != nil {
		return 0, 0, err
	}

	// For simplicity, we'll return dummy partition and offset
	// In production, you'd want to get these from the WriteMessages response
	return 0, time.Now().Unix(), nil
}

// Close closes all Kafka connections
func (es *EventService) Close() {
	es.cancel()
	es.wg.Wait()

	for _, producer := range es.producers {
		producer.Close()
	}

	for _, consumer := range es.consumers {
		consumer.Close()
	}
}

// HTTP Handlers
func (es *EventService) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"status": true})
}

func (es *EventService) handleMovieEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var movieEvent MovieEvent
	if err := json.NewDecoder(r.Body).Decode(&movieEvent); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	event := Event{
		ID:        uuid.New().String(),
		Type:      "movie",
		Timestamp: time.Now(),
		Payload:   movieEvent,
	}

	partition, offset, err := es.PublishEvent("movie-events", event)
	if err != nil {
		log.Printf("Error publishing movie event: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("[PRODUCER] Published movie event: %s, Action: %s, MovieID: %d",
		event.ID, movieEvent.Action, movieEvent.MovieID)

	response := EventResponse{
		Status:    "success",
		Partition: partition,
		Offset:    offset,
		Event:     event,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (es *EventService) handleUserEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var userEvent UserEvent
	if err := json.NewDecoder(r.Body).Decode(&userEvent); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Set timestamp if not provided
	if userEvent.Timestamp.IsZero() {
		userEvent.Timestamp = time.Now()
	}

	event := Event{
		ID:        uuid.New().String(),
		Type:      "user",
		Timestamp: time.Now(),
		Payload:   userEvent,
	}

	partition, offset, err := es.PublishEvent("user-events", event)
	if err != nil {
		log.Printf("Error publishing user event: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("[PRODUCER] Published user event: %s, Action: %s, UserID: %d",
		event.ID, userEvent.Action, userEvent.UserID)

	response := EventResponse{
		Status:    "success",
		Partition: partition,
		Offset:    offset,
		Event:     event,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (es *EventService) handlePaymentEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var paymentEvent PaymentEvent
	if err := json.NewDecoder(r.Body).Decode(&paymentEvent); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Set timestamp if not provided
	if paymentEvent.Timestamp.IsZero() {
		paymentEvent.Timestamp = time.Now()
	}

	event := Event{
		ID:        uuid.New().String(),
		Type:      "payment",
		Timestamp: time.Now(),
		Payload:   paymentEvent,
	}

	partition, offset, err := es.PublishEvent("payment-events", event)
	if err != nil {
		log.Printf("Error publishing payment event: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("[PRODUCER] Published payment event: %s, Status: %s, PaymentID: %d, Amount: %.2f",
		event.ID, paymentEvent.Status, paymentEvent.PaymentID, paymentEvent.Amount)

	response := EventResponse{
		Status:    "success",
		Partition: partition,
		Offset:    offset,
		Event:     event,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func main() {
	// Get Kafka brokers from environment
	kafkaBrokers := os.Getenv("KAFKA_BROKERS")
	if kafkaBrokers == "" {
		kafkaBrokers = "localhost:9092"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	log.Printf("Starting Events Service")
	log.Printf("Kafka Brokers: %s", kafkaBrokers)
	log.Printf("Port: %s", port)

	// Create event service
	eventService := NewEventService([]string{kafkaBrokers})
	defer eventService.Close()

	// Setup HTTP routes
	http.HandleFunc("/api/events/health", eventService.handleHealth)
	http.HandleFunc("/api/events/movie", eventService.handleMovieEvent)
	http.HandleFunc("/api/events/user", eventService.handleUserEvent)
	http.HandleFunc("/api/events/payment", eventService.handlePaymentEvent)

	// Setup graceful shutdown
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      nil,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Events service listening on port %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
