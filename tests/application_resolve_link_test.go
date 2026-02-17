package tests

import (
	"testing"
	"time"

	resolve_link2 "github.com/n1ckerr0r/shortener/core/application/resolve_link"
	"github.com/n1ckerr0r/shortener/core/domain/link"
	"github.com/n1ckerr0r/shortener/infrastructure/repository"
)

func TestService_Resolve_Success(t *testing.T) {
	repo := repository.NewMemoryRepository()
	now := time.Now()

	// Создаем правильные значения для ссылки
	shortCode, err := link.NewShortCode("abc123")
	if err != nil {
		t.Fatal(err)
	}

	originalURL, err := link.NewOriginalURL("https://example.com")
	if err != nil {
		t.Fatal(err)
	}

	expiration := link.NewExpiration(nil) // nil означает без срока действия

	linkObj, err := link.NewShortLink(
		shortCode,
		originalURL,
		now,
		expiration,
	)
	if err != nil {
		t.Fatal(err)
	}

	if err = repo.Save(linkObj); err != nil {
		t.Fatal(err)
	}

	clock := fakeClock{now: now}
	service := resolve_link2.NewService(repo, clock)

	resp, err := service.Resolve(resolve_link2.Request{
		Code: "abc123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.OriginalURL != "https://example.com" {
		t.Fatalf("unexpected url: %s", resp.OriginalURL)
	}
}

func TestService_Resolve_Expired(t *testing.T) {
	repo := repository.NewMemoryRepository()

	// Создаем ссылку с сроком действия +1 час от текущего времени
	creationTime := time.Now()
	expirationTime := creationTime.Add(time.Hour) // срок действия через час

	shortCode, err := link.NewShortCode("abc123")
	if err != nil {
		t.Fatal(err)
	}

	originalURL, err := link.NewOriginalURL("https://example.com")
	if err != nil {
		t.Fatal(err)
	}

	expiration := link.NewExpiration(&expirationTime)

	linkObj, err := link.NewShortLink(
		shortCode,
		originalURL,
		creationTime,
		expiration,
	)
	if err != nil {
		t.Fatal(err)
	}

	if err = repo.Save(linkObj); err != nil {
		t.Fatal(err)
	}

	// Теперь используем время ПОСЛЕ истечения срока действия
	currentTime := expirationTime.Add(time.Minute) // через минуту после истечения

	clock := fakeClock{now: currentTime}
	service := resolve_link2.NewService(repo, clock)

	_, err = service.Resolve(resolve_link2.Request{
		Code: "abc123",
	})

	if err == nil {
		t.Fatal("expected error for expired link")
	}

	// Проверяем, что это именно ошибка об истечении срока
	expectedErr := "link is expired" // или конкретная ошибка из вашего кода
	if err.Error() != expectedErr {
		t.Fatalf("expected '%s' error, got '%v'", expectedErr, err)
	}
}
