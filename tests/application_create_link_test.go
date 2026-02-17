package tests

import (
	"testing"
	"time"

	create_link2 "github.com/n1ckerr0r/shortener/core/application/create_link"
	"github.com/n1ckerr0r/shortener/core/domain/link"
)

type FakeRepo struct {
	saved bool
	link  *link.ShortLink
}

func (f *FakeRepo) Save(l *link.ShortLink) error {
	f.saved = true
	f.link = l
	return nil
}

func (f *FakeRepo) Find(code link.ShortCode) (*link.ShortLink, error) {
	return nil, nil
}

func (f *FakeRepo) Exists(code link.ShortCode) (bool, error) {
	return false, nil
}

func (f *FakeRepo) Delete(code link.ShortCode) error {
	return nil
}

type FakeGenerator struct{}

func (f *FakeGenerator) Generate() (link.ShortCode, error) {
	return link.NewShortCode("abc123")
}

type FakeClock struct{}

func (f *FakeClock) Now() time.Time {
	return time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
}

func TestCreateShortLink_Success(t *testing.T) {
	repo := &FakeRepo{}
	gen := &FakeGenerator{}
	clock := &FakeClock{}
	service := create_link2.NewService(repo, gen, clock)

	req := create_link2.Request{
		OriginalURL: "https://example.com",
	}

	resp, err := service.CreateShortLink(req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.ShortCode != "abc123" {
		t.Fatalf("expected abc123, got %s", resp.ShortCode)
	}

	if !repo.saved {
		t.Fatal("link was not saved")
	}
}

func TestCreateShortLink_InvalidURL(t *testing.T) {
	repo := &FakeRepo{}
	gen := &FakeGenerator{}
	clock := &FakeClock{}

	service := create_link2.NewService(repo, gen, clock)

	req := create_link2.Request{
		OriginalURL: "not a url",
	}

	_, err := service.CreateShortLink(req)

	if err == nil {
		t.Fatal("expected error")
	}

	if repo.saved {
		t.Fatal("repo.Save should not be called")
	}
}
