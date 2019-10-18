package kube

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/sighupio/permission-manager/server/kube"

	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)


func CreateKubeconfigYAML(username string) {
	rsaFile, err := ioutil.TempFile(os.TempDir(), "prefix-")
	if err != nil {
		log.Fatal("Cannot create temporary file", err)
	}
	defer os.Remove(rsaFile.Name())

	rsaPrivateKey, err := exec.Command("openssl", "genrsa", "4096").Output()
	if err != nil {
		panic(err)
	}

	if _, err = rsaFile.Write(rsaPrivateKey); err != nil {
		log.Fatal("Failed to write to temporary file", err)
	}

	subj := fmt.Sprintf("/CN=%s", username)
	cmd := exec.Command("openssl", "req", "-new", "-key", rsaFile.Name(), "-subj", subj)
	csr, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(fmt.Sprint(err))
	}

	clientCsrFile, err := ioutil.TempFile(os.TempDir(), "prefix-")
	if err != nil {
		log.Fatal("Cannot create temporary file", err)
	}
	if _, err = clientCsrFile.Write(csr); err != nil {
		log.Fatal("Failed to write to temporary file", err)
	}
	defer os.Remove(clientCsrFile.Name())
	crt, err := exec.Command("openssl", "x509", "-req", "-days", "365", "-sha256",
		"-in",
		clientCsrFile.Name(),
		"-CA",
		filepath.Join(os.Getenv("HOME"), ".minikube", "ca.crt"),
		"-CAkey",
		filepath.Join(os.Getenv("HOME"), ".minikube", "ca.key"),
		"-set_serial",
		"2",
	).Output()
	if err != nil {
		panic(err)
	}

	clusterName := "minikube"
	cacertPath := filepath.Join(os.Getenv("HOME"), ".minikube", "ca.crt")

	crtBase64 := base64.StdEncoding.EncodeToString(crt)
	rsaPrivateKeyBase64 := base64.StdEncoding.EncodeToString(rsaPrivateKey)

	kubeconfig := fmt.Sprintf(`apiVersion: v1
kind: Config
preferences:
    colors: true
current-context: %s
clusters:
  - name: %s
    cluster:
      server: https://192.168.99.100:8443
      certificate-authority: %s}
contexts:
  - context:
      cluster: %s
      user: %s
    name: %s
users:
  - name: %s
    user:
      client-certificate-data: %s
      client-key-data: %s`,
		clusterName, clusterName, cacertPath, clusterName, username, clusterName, username, crtBase64, rsaPrivateKeyBase64)

}