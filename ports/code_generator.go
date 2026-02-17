package ports

import "github.com/n1ckerr0r/shortener/core/domain/link"

type CodeGenerator interface {
	Generate() (link.ShortCode, error)
}
