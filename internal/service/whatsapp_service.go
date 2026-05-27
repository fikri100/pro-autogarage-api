package service

import (
	"context"
	"log"
)

// WhatsAppService is the interface for sending WhatsApp messages
type WhatsAppService interface {
	SendOTP(ctx context.Context, phone string, otp string) error
}

// MockWhatsAppService is a mock implementation of WhatsAppService for development
type MockWhatsAppService struct{}

// NewMockWhatsAppService creates a new MockWhatsAppService
func NewMockWhatsAppService() *MockWhatsAppService {
	return &MockWhatsAppService{}
}

// SendOTP simulates sending an OTP via WhatsApp by printing to standard logs
func (s *MockWhatsAppService) SendOTP(ctx context.Context, phone string, otp string) error {
	log.Printf("\n============================================================\n[MOCK WHATSAPP API] MENGIRIM KODE OTP [%s] KE NOMOR WA: %s\n============================================================", otp, phone)
	return nil
}
