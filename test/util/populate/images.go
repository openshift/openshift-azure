package populate

import "fmt"

// DummyImagePullSecret return dummy ImagePullSecrets with the registry as key
func DummyImagePullSecret(registry string) []byte {
	return []byte(fmt.Sprintf("{\"auths\": {\"%s\": {\"auth\": \"dGVzdDp0ZXN0Cg==\"}}}", registry))
}
