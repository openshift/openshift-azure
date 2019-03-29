package keylog

import (
	"bytes"
	"io"
	"os"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
	"golang.org/x/crypto/openpgp/packet"
)

var (
	f          *os.File
	wc         io.WriteCloser
	ciphertext io.WriteCloser
)

func init() {
	public := []byte(`-----BEGIN PGP PUBLIC KEY BLOCK-----

mQENBE7GXFQBCADX8nRFcANMyvRHtlc4v1g+9UCxprb6tQchdulT+md2JpSa7X2U
xhccL3hLRIa+z/waNwG6Pso4IFyhwxkZaXCC/vk7P6gsyn5S3Q8yg4j5FE8vM5Ct
DoPGNJzlC0ku1IogvuWS0IB5JJBLjHLbILOFst/PTAQm92UmZ1fU+AJHKLQU9pVQ
j9gCsVWE3r4mJbgf1LhrXfmIOo5OMsXUxHSuWN+tfFM51k7ltKEyEGaH1yG8Wvol
2fQZuSl929TnjnRIsSSK/gohE5+UJ737QpyGFE95pR7qLnjnDAVxR+R9w3pEvNBE
fH4PK9nNW8j2wVcGBqcp9F0CFkliGD22lh5rABEBAAG0H0ppbSBNaW50ZXIgPGpt
aW50ZXJAcmVkaGF0LmNvbT6JATgEEwECACIFAk7GXFQCGwMGCwkIBwMCBhUIAgkK
CwQWAgMBAh4BAheAAAoJEAcwy9oQ0aLTYUsH/3NSCfpuveVMUY5oCYIJB1tWelGS
J/zSK+SXqotHsEynFkvgUk4DBhs7I73MN7jK6kOzCKuIPwGo9pgMUtQiG1KILtCM
ueI+cwgGtQLX8Q2Py3g3bNRjMObbprHCpA/DegTFAz8ibqeCGq0xRpFUj/toLeHS
E4H+3jp60E0LtG93TVz+EMnf0jZiPlCGwQ9eIWPxjpT6kkqbBDknx9ge6ZlWEmtn
3j5e7fcKYN9chLlqqvt11IpKJjc5kv9JSjA755OCZGbgh5+jBBhylpXT8KOpR5eG
Z7afAltjiAE2ccK5UTwM5xiKmR9zuTpHypW3vC++W7PEd7pOHcTSbAYk6Fe5AQ0E
TsZcVAEIANgH0jJ1M3r3gEViiV3y6OxrLv/K561yOtHFnQ1WJXr7QKJTDtPZgXhF
eMjzVbrrXy8sBwd9CfQH4Ghnz/WtHX/D5B57auwKBEawWtHSivouOPyxqkql2quR
e1itSrfWTjMeJmvoblHZ6zRjqQrpVfizJqgR7/zeIFN84Rjx2kZzvwAD6aT0fba5
yYhMuFK6k0AbsYC3MpWLrMyrI6GWQB2yFNwsVE5JnfM2dgJwPnSCFwqwd7cDEgKR
30v5lypAaMLX6hGaC/s/Vacmm03DtH1sPfRpIGuZpsKGzrp+jA09q9rqpeAW1yw9
ykQB3mn3ddZ8k18wDZjX6oerm94eXYkAEQEAAYkBHwQYAQIACQUCTsZcVAIbDAAK
CRAHMMvaENGi063eB/0cOtYTijpvlg5iXcyK86ysEMO5OZX2v5wBKx2dw21QunaJ
d7CerU+BUOOjR4vT34t8Gup+8EMX1XNWhXWEK0/oI9t0k1HfIdZl/0fcJfucRJ+a
uIBwBxYspfXerDOJcw5IBYY3IwiuyuvYSU7GTDFQ6ant0BimRgWBpEeeXbaXvcVI
y/WdGdSi4Gq8g3NpdwWr0zc7QyIcFjRWmbK+xljz5vq3tr6LbX9dPyWPwjJD2g4/
YA/vUI3swZYxFhi5rR561mibXQ4w2NkUe2stTR/fQ/Xp2hljnn9u36obibKY3zVz
Jm63Tap+BoOQgV9umzXD6KXEYjyUEogi5od40Hi/
=mTSd
-----END PGP PUBLIC KEY BLOCK-----
`)

	block, err := armor.Decode(bytes.NewReader(public))
	if err != nil {
		panic(err)
	}

	if block.Type != openpgp.PublicKeyType {
		panic(err)
	}

	entity, err := openpgp.ReadEntity(packet.NewReader(block.Body))
	if err != nil {
		panic(err)
	}

	err = os.MkdirAll("/tmp/artifacts", 0777)
	if err != nil {
		panic(err)
	}

	f, err = os.Create("/tmp/artifacts/keylog")
	if err != nil {
		panic(err)
	}

	ciphertext, err = armor.Encode(f, "PGP MESSAGE", nil)
	if err != nil {
		panic(err)
	}

	wc, err = openpgp.Encrypt(ciphertext, []*openpgp.Entity{entity}, nil, nil, nil)
	if err != nil {
		panic(err)
	}
}

func Writer() io.Writer {
	return wc
}

func Done() {
	err := wc.Close()
	if err != nil {
		panic(err)
	}

	err = ciphertext.Close()
	if err != nil {
		panic(err)
	}

	_, err = f.WriteString("\n")
	if err != nil {
		panic(err)
	}

	err = f.Close()
	if err != nil {
		panic(err)
	}
}
