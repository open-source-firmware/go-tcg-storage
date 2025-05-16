package cmdutil

import (
	"fmt"

	"github.com/open-source-firmware/go-tcg-storage/pkg/core"
	"github.com/open-source-firmware/go-tcg-storage/pkg/core/hash"
)

type PasswordEmbed struct {
	Password string `required:"" env:"PASS" type:"password" help:"Authentication password"`
	Hash     string `optional:"" env:"HASH" default:"dta" enum:"sedutil-dta,sedutil-sha512,dta,sha1,sha512" help:"Use dta (sha1) or sha512 for password hashing"`
}

func (t *PasswordEmbed) GenerateHash(coreObj *core.Core) ([]byte, error) {
	serial, err := coreObj.SerialNumber()
	if err != nil {
		return nil, fmt.Errorf("coreObj.SerialNumber() failed: %v", err)
	}
	salt := string(serial)

	switch t.Hash {
	// Drive-Trust-Alliance uses sha1
	case "sedutil-dta", "sha1", "dta":
		return hash.HashSedutilDTA(t.Password, salt), nil
	// ChubbyAnt uses sha512
	case "sedutil-sha512", "sha512":
		return hash.HashSedutil512(t.Password, salt), nil
	default:
		return nil, fmt.Errorf("unknown hash method %q", t.Hash)
	}
}
