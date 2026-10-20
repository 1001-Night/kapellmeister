// SPDX-License-Identifier: MPL-2.0
// Certificate generation and kubeconfig assembly adapted from:
// terraform-provider-openstack/openstack/containerinfra_shared_v1.go
// https://github.com/terraform-provider-openstack/terraform-provider-openstack/
// blob/1268c8db29c9e296200b9f1d92e6dea92eed33a4/openstack/containerinfra_shared_v1.go

package openstack

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/containerinfra/v1/certificates"
	"github.com/gophercloud/gophercloud/v2/openstack/containerinfra/v1/clusters"
	"github.com/gophercloud/utils/v2/openstack/clientconfig"
	"gopkg.in/yaml.v3"
)

// Выбирает профиль или авторизацию из OS_*.
func clientOptions(cloud, region, endpoint string) (*clientconfig.ClientOpts, error) {
	if cloud == "" {
		cloud = os.Getenv("OS_CLOUD")
	}
	if cloud == "" && os.Getenv("OS_AUTH_URL") == "" {
		clouds, err := clientconfig.LoadCloudsYAML()
		if err != nil {
			return nil, fmt.Errorf("load default clouds.yaml (or export OS_AUTH_URL and OS_* credentials): %w", err)
		}
		if len(clouds) != 1 {
			names := make([]string, 0, len(clouds))
			for name := range clouds {
				names = append(names, name)
			}
			sort.Strings(names)
			return nil, fmt.Errorf("select a cloud with --cloud or OS_CLOUD; available: %v", names)
		}
		for name := range clouds {
			cloud = name
		}
	}
	return &clientconfig.ClientOpts{Cloud: cloud, RegionName: region, EndpointType: endpoint}, nil
}

// Создаёт клиент Magnum из clouds.yaml или OS_*.
func NewMagnumClient(ctx context.Context, cloud, region, endpoint string) (*gophercloud.ServiceClient, error) {
	opts, err := clientOptions(cloud, region, endpoint)
	if err != nil {
		return nil, err
	}
	return clientconfig.NewServiceClient(ctx, "container-infra", opts)
}

// Собирает kubeconfig. Каждый вызов выпускает новый клиентский сертификат.
func Kubeconfig(ctx context.Context, client *gophercloud.ServiceClient, clusterID string) ([]byte, error) {
	if strings.TrimSpace(clusterID) == "" {
		return nil, errors.New("cluster name or UUID is required")
	}
	cluster, err := clusters.Get(ctx, client, clusterID).Extract()
	if err != nil {
		return nil, fmt.Errorf("get cluster: %w", err)
	}
	if cluster.UUID == "" || cluster.APIAddress == "" {
		return nil, errors.New("cluster UUID or api_address is empty; check cluster status")
	}
	ca, err := certificates.Get(ctx, client, cluster.UUID).Extract()
	if err != nil {
		return nil, fmt.Errorf("get cluster CA: %w", err)
	}
	key, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, fmt.Errorf("generate private key: %w", err)
	}
	csr, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{
		SignatureAlgorithm: x509.SHA512WithRSA,
		Subject:            pkix.Name{CommonName: "admin", Organization: []string{"system:masters"}},
	}, key)
	if err != nil {
		return nil, fmt.Errorf("generate CSR: %w", err)
	}
	cert, err := certificates.Create(ctx, client, certificates.CreateOpts{
		ClusterUUID: cluster.UUID,
		CSR:         string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csr})),
	}).Extract()
	if err != nil {
		return nil, fmt.Errorf("sign client certificate: %w", err)
	}
	if ca.PEM == "" || cert.PEM == "" {
		return nil, errors.New("Magnum returned an empty CA or client certificate")
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	name := cluster.Name
	if name == "" {
		name = cluster.UUID
	}
	user := name + "-admin"
	b64 := func(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }
	return yaml.Marshal(map[string]any{
		"apiVersion":      "v1",
		"kind":            "Config",
		"current-context": name,
		"clusters": []any{map[string]any{
			"name":    name,
			"cluster": map[string]string{"server": cluster.APIAddress, "certificate-authority-data": b64(ca.PEM)},
		}},
		"contexts": []any{map[string]any{
			"name": name, "context": map[string]string{"cluster": name, "user": user},
		}},
		"users": []any{map[string]any{
			"name": user,
			"user": map[string]string{"client-certificate-data": b64(cert.PEM), "client-key-data": b64(string(keyPEM))},
		}},
	})
}
