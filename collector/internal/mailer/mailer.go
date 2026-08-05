package mailer

import (
	"errors"

	"github.com/resend/resend-go/v2"
)

// Service é a estrutura que encapsula o motor de envio
type Service struct {
	client *resend.Client
	from   string
}

// New cria uma nova instância do serviço de e-mail
func New(apiKey string) (*Service, error) {
	if apiKey == "" {
		return nil, errors.New("chave da API do Resend não fornecida")
	}

	return &Service{
		client: resend.NewClient(apiKey),
		// Em modo de testes, o Resend exige que o remetente seja este:
		from: "Lume <onboarding@resend.dev>",
	}, nil
}

// Send dispara o e-mail transacional
func (s *Service) Send(to string, subject string, htmlBody string) error {
	params := &resend.SendEmailRequest{
		From:    s.from,
		To:      []string{to}, // Podemos adicionar mais destinatários no array se precisar
		Subject: subject,
		Html:    htmlBody,
	}

	_, err := s.client.Emails.Send(params)
	return err
}
