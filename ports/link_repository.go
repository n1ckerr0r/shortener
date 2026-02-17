package ports

import "github.com/n1ckerr0r/shortener/core/domain/link"

type LinkRepository interface {
	Save(link *link.ShortLink) error
	Find(code link.ShortCode) (*link.ShortLink, error)
	Exists(code link.ShortCode) (bool, error)
	//Delete(code link.ShortCode) error
}
