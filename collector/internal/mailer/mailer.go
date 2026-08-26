package mailer

import (
	"errors"
	"log/slog"

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

	// Dispara o e-mail e captura a resposta da API (que contém o ID de rastreio)
	resp, err := s.client.Emails.Send(params)
	if err != nil {
		// [SLOG] Falha na comunicação com a API do Resend (Timeout, API Key inválida, etc)
		slog.Error("erro na api do resend ao enviar email",
			slog.String("event", "resend_api_error"),
			slog.String("to", to),
			slog.String("subject", subject),
			slog.String("error", err.Error()),
		)
		return err
	}

	// [SLOG] Sucesso na entrega para o Resend.
	// Salvar o resp.Id é vital para rastrear bounce/spam no painel do Resend depois.
	slog.Info("email processado pelo resend",
		slog.String("event", "resend_api_success"),
		slog.String("to", to),
		slog.String("subject", subject),
		slog.String("resend_message_id", resp.Id),
	)

	return nil
}
