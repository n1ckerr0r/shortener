package link

type ShortCode struct {
	value string
}

func NewShortCode(code string) (ShortCode, error) {
	if code == "" {
		return ShortCode{}, ErrEmptyShortCode
	}

	return ShortCode{value: code}, nil
}

func (sc ShortCode) Value() string {
	return sc.value
}
