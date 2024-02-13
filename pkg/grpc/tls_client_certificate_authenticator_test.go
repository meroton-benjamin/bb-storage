package grpc_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/buildbarn/bb-storage/internal/mock"
	"github.com/buildbarn/bb-storage/pkg/auth"
	bb_grpc "github.com/buildbarn/bb-storage/pkg/grpc"
	auth_pb "github.com/buildbarn/bb-storage/pkg/proto/auth"
	"github.com/buildbarn/bb-storage/pkg/testutil"
	"github.com/golang/mock/gomock"
	"github.com/jmespath/go-jmespath"
	"github.com/stretchr/testify/require"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

var (
	// Certificates generated by running:
	// openssl req -x509 -sha256 -nodes -days 365 -newkey rsa:4096 -keyout key -out crt -extensions extensions -config tls_client_certificate_valid.cnf
	certificateValid = parseCertificate(`
-----BEGIN CERTIFICATE-----
MIIFDTCCAvWgAwIBAgIJAOfy0XE0ATEbMA0GCSqGSIb3DQEBCwUAMBgxFjAUBgNV
BAMMDWEuZXhhbXBsZS5jb20wHhcNMjQwNDA5MjAxODAxWhcNMjUwNDA5MjAxODAx
WjAYMRYwFAYDVQQDDA1hLmV4YW1wbGUuY29tMIICIjANBgkqhkiG9w0BAQEFAAOC
Ag8AMIICCgKCAgEAwLWyeSp8QaCbJ7+a8IJFqv7/3Zx/Ie8lExWSOR1sfQD8TC40
E5Tj/lNgFxyImjlgVray4iYaHeR35BYKi6EZud5J1TG2NJQfoRb4GVYxwDYn0A06
+DSGMLLhAVvzFuXGpW0aOD0L9CJDqtG1HQEbqktppGgxV148AvEse4ZqfOm1XbGc
tdY2+bLR76YNycN59RrGs9n6C9SjJ7cxf+/DJbEgzGLK0zw07hou+oAnZJs3g8hD
N27F+m9hxf486rSnJMb+SFS4Clm6d0SFDK0hsleeRTzsHfxQXpX5LMjNyGvQ6wdk
nIrIOFluIx2AoRiC2HqJYfVREsV1cCLjp88gg0flqP6xW8Hz7ThsEAv1lomzeKbV
7nmLaZxWaIUHPLtvc1/ZKw2hmfjbJllVUD/Vrg3V1XRb/WFJ2gBlddf2kcyQYkXX
DTpDtaGAw0xgirtz8K7pWdtKiF5a8LOFLr+Y8GGz67hD/l7D9I0PkSPWq44Xx65v
ZOylcS7NXJUq07K5si2CqA3ga83EOZvErTj3KDA0lgZR4oLrZN7qjYkrmXz6d2nh
i1Tciz5O/d+YHBK5+5/nZashI8zZsyqMd9FeXgSEWfqWKTkL4UDY3fr9ML7TZbY7
tumPJfRy5XzPeFM+ctcOMfpsuJmVIKrd+GDjW9CDFt2PzR4Z6+kOVf3f3JcCAwEA
AaNaMFgwHQYDVR0lBBYwFAYIKwYBBQUHAwEGCCsGAQUFBwMCMDcGA1UdEQQwMC6C
DWEuZXhhbXBsZS5jb22BDm1lQGV4YW1wbGUuY29thg11cmk6ZXhhbXBsZTphMA0G
CSqGSIb3DQEBCwUAA4ICAQBHh03FtmDdPv2/afrt6TZeOg7YNCaVSax34kuqDC3f
Hn9bSz2f+qB2frGh2q0dNBu5LYNXLVsOP/Whl5tSc5CuvspoC/otB1qfJkGPa2lz
wJ4yaWuUwsghh98fl5+OBM12hEvXVH500D22y12sR+BuJk34skZrvaLELNhFM6il
ek9J8Aas0DZi4g9Kv7QKP4cFdc6cIU4ubebE0IGHOfAhRn7nJYJNFq415DQcoBQ/
qsl7jydKzwJSUkt6H+M9WUAi+DqvuEWK2hi9wX4KxykFTC9gRgaqyr7Xnsx/JfEO
Fp+tqQ1aH/pfIf4XtC0u/qxV8mi5Hfpi5gmid/XnoHSaaJsf4JXSVSkOFguH08+h
NDOJwTlL+ukiIF4ZXxw6wdCVk6za8B0POq9USg29MCJOSNLHO/3RMULMfeVW9dys
X45aMdj6OuUfJHL8v8tOucfLuG10068speNncnn9UZqoe2TAGU0ePjFlNTS7w2vo
SSqmKrEFbn9Kg9i0woGcTzgvqFuX6Dajl1pKHNKrsDEy0RInIctfOnZq8h4xJ3Pv
d3AogMA0xXQCbd+yj/cUSQvcqGPeHPGRsd0WLFqxGDCJawg7hR5fKQHWhew/Xy0R
lMZj8cfXAlpFnkEWirjXSuc5sD2k9/98pSgNOHsW7VVKIQVOmbdDvx9HiViNTtLp
Ww==
-----END CERTIFICATE-----`)
	certificateUnrelated = parseCertificate(`
-----BEGIN CERTIFICATE-----
MIIE7jCCAtagAwIBAgIJANGCy1pXrDH2MA0GCSqGSIb3DQEBCwUAMBgxFjAUBgNV
BAMMDWIuZXhhbXBsZS5jb20wHhcNMjQwNDA5MjAwNTAxWhcNMjUwNDA5MjAwNTAx
WjAYMRYwFAYDVQQDDA1iLmV4YW1wbGUuY29tMIICIjANBgkqhkiG9w0BAQEFAAOC
Ag8AMIICCgKCAgEAw76FLvERkwdUZ6pTIQaXOmtkgNqTLKWTnw/3EVCDhX6BKSsV
PA1bd09QUWTjqG93zwREML+q+uFg1C1UHpUH18SQ5xGtd8XvPhnI5DV3zYfyY2K8
7u7Umqp2ozID6w6X2ZynzF05U4DtpJ9vYWMsIylRHDmA9all+LNKDJQMnKu/ChWU
wVczjKXCMbxyMQ163Cur73R7iKT3Hdlsxxv+3dO5Qz+tWsl2mF58LIZh4uGeeIZo
UGkK+2TAA3aPfYWoSSs0molztNkq2phxvl3Ctw2P3SJs3Pf2v8t5u9iAPPOLmPZL
M1wASZFCdzZ+Jv2/JyD+j36LKowBem7N+IhXydCvVEX7gZcBsxLlTRCKAGoHgeiE
dD6i9M+0esEZgsH+DxHn92/4IwRRR8KRzQGclfwBC9dFi6rdviHMjAX7GGlrgapO
pw1fsf+UwGpuDy9skeu5SN3P1PhFIHVzuy3oFMuJgnalIjOv5WTLW/WOJ4DXkqEX
TQ/yOdmW9nxiWpU0I6LSHbjLP9vPcGp7dWVvvV9i80JoRIEpPywkEj3+SZwm9m1c
1mACkyQ852xT7v01AgZkbXRlTRR72vUcvgGHObKgvvvQUwexP8Vt1IMDjNM3dUS1
iZc+0XYCs/01bbzFMEhQa1qZTmJdQsahE9gWJ+/858aJVFMo7U3OTSRQXGMCAwEA
AaM7MDkwHQYDVR0lBBYwFAYIKwYBBQUHAwEGCCsGAQUFBwMCMBgGA1UdEQQRMA+C
DWIuZXhhbXBsZS5jb20wDQYJKoZIhvcNAQELBQADggIBADF/3M2UCjdKQ2Ht9jQY
rPzpM3GsyXCnEST16/RTCTGDmn9gLw0gHMOHRjAUpsRQ148sJvFPuDhL/LnPzxwU
rNxCyQj3tnh6ye9/2XStRwUor22LvJDMItlAMON6ifolh90fD/0MPFB3Iw0jbv2b
c9AXZiVaj5uFfyeL9fi53eoK0UXwOoWjVJeTvTlW/976CncqAtW8dWUo9xp7Ajdh
0J/0frkB4oVN448XD/KM4a0kgFmK8dCLEtwjY9cFJGqvsnvLOyAylvdMT6sXd6BB
x3SuK8kPpTliqgVfElmy88UPpUNyIVBFq6gR+a30BfoM2Yrh84FKAJHGhKR1XSMq
oeayLrdzOzxOBYHuEii3wYHMvc+f30tfGLWcCAyvYtrXHtBY2u87nZjK04ASt9+9
Abk3O8CTwM49b1P60b2F9GOxXT6K69JVRrJqaUKNUDiOKu9K7RQ9DSHLEy8W4LVZ
l+UtN27XkH+q8YltubjDdv84HAA/9CgiXbVlIeVLCmIehPaofLz9MFf10HTDz/gn
hMLjxsaanOX+AVRcIDooJg6w05P3SyKz5NR8C7j69JGNNiucrOfFByQ45rzMjjzS
mDFvCpPgwj29k6xjxCwFNgI/9wEwq5BaOtZ5bm6RgeqGE63RgsZYIfNVfOML0gV1
zz99b0I42KpaTWavqKDevUg6
-----END CERTIFICATE-----`)
)

func parseCertificate(v string) *x509.Certificate {
	block, _ := pem.Decode([]byte(v))
	certificate, _ := x509.ParseCertificate(block.Bytes)
	return certificate
}

func TestTLSClientCertificateAuthenticator(t *testing.T) {
	ctrl, ctx := gomock.WithContext(context.Background(), t)

	clientCAs := x509.NewCertPool()
	clientCAs.AddCert(certificateValid)
	clock := mock.NewMockClock(ctrl)
	expectedMetadata := auth.MustNewAuthenticationMetadataFromProto(&auth_pb.AuthenticationMetadata{
		Public: structpb.NewStructValue(&structpb.Struct{
			Fields: map[string]*structpb.Value{
				"dnsNames": structpb.NewListValue(&structpb.ListValue{
					Values: []*structpb.Value{
						structpb.NewStringValue("a.example.com"),
					},
				}),
				"emailAddresses": structpb.NewListValue(&structpb.ListValue{
					Values: []*structpb.Value{
						structpb.NewStringValue("me@example.com"),
					},
				}),
				"uris": structpb.NewListValue(&structpb.ListValue{
					Values: []*structpb.Value{
						structpb.NewStringValue("uri:example:a"),
					},
				}),
			},
		}),
	})

	aValidator := jmespath.MustCompile(`contains(dnsNames, 'a.example.com')`)
	bValidator := jmespath.MustCompile(`contains(dnsNames, 'b.example.com')`)
	metadataExtractor := jmespath.MustCompile(`{"public": @}`)
	aAuthenticator := bb_grpc.NewTLSClientCertificateAuthenticator(clientCAs, clock, aValidator, metadataExtractor)
	bAuthenticator := bb_grpc.NewTLSClientCertificateAuthenticator(clientCAs, clock, bValidator, metadataExtractor)

	t.Run("NoGRPC", func(t *testing.T) {
		// Authenticator is used outside of gRPC, meaning it cannot
		// extract peer state information.
		_, err := aAuthenticator.Authenticate(ctx)
		testutil.RequireEqualStatus(
			t,
			status.Error(codes.Unauthenticated, "Connection was not established using gRPC"),
			err)
	})

	t.Run("NoTLS", func(t *testing.T) {
		// Non-TLS connection.
		_, err := aAuthenticator.Authenticate(peer.NewContext(ctx, &peer.Peer{}))
		testutil.RequireEqualStatus(
			t,
			status.Error(codes.Unauthenticated, "Connection was not established using TLS"),
			err)
	})

	t.Run("NoCertificateProvided", func(t *testing.T) {
		// Connection with no certificate provided by the client.
		_, err := aAuthenticator.Authenticate(
			peer.NewContext(
				ctx,
				&peer.Peer{
					AuthInfo: credentials.TLSInfo{
						State: tls.ConnectionState{},
					},
				}))
		testutil.RequireEqualStatus(
			t,
			status.Error(codes.Unauthenticated, "Client provided no TLS client certificate"),
			err)
	})

	t.Run("NoCAMatch", func(t *testing.T) {
		// Connection with a certificate that doesn't match the CA.
		clock.EXPECT().Now().Return(time.Unix(1712700000, 0))
		_, err := aAuthenticator.Authenticate(
			peer.NewContext(
				ctx,
				&peer.Peer{
					AuthInfo: credentials.TLSInfo{
						State: tls.ConnectionState{
							PeerCertificates: []*x509.Certificate{
								certificateUnrelated,
							},
						},
					},
				}))
		testutil.RequireEqualStatus(
			t,
			status.Error(codes.Unauthenticated, "Cannot validate TLS client certificate: x509: certificate signed by unknown authority"),
			err)
	})

	t.Run("Expired", func(t *testing.T) {
		// Connection with a certificate that is signed by the
		// right CA, but expired.
		clock.EXPECT().Now().Return(time.Unix(1750000000, 0))
		_, err := aAuthenticator.Authenticate(
			peer.NewContext(
				ctx,
				&peer.Peer{
					AuthInfo: credentials.TLSInfo{
						State: tls.ConnectionState{
							PeerCertificates: []*x509.Certificate{
								certificateValid,
							},
						},
					},
				}))
		testutil.RequireEqualStatus(
			t,
			status.Error(codes.Unauthenticated, "Cannot validate TLS client certificate: x509: certificate has expired or is not yet valid: current time 2025-06-15T15:06:40Z is after 2025-04-09T20:18:01Z"),
			err)
	})

	t.Run("ValidationFail", func(t *testing.T) {
		// Connection with a certificate that is signed by the
		// right CA, but expired.
		clock.EXPECT().Now().Return(time.Unix(1712700000, 0))

		_, err := bAuthenticator.Authenticate(
			peer.NewContext(
				ctx,
				&peer.Peer{
					AuthInfo: credentials.TLSInfo{
						State: tls.ConnectionState{
							PeerCertificates: []*x509.Certificate{
								certificateValid,
							},
						},
					},
				}))
		testutil.RequireEqualStatus(
			t,
			status.Error(codes.Unauthenticated, "Rejected TLS client certificate claims"),
			err)
	})

	t.Run("Success", func(t *testing.T) {
		// Connection with at least one verified chain.
		clock.EXPECT().Now().Return(time.Unix(1712700000, 0))
		actualMetadata, err := aAuthenticator.Authenticate(
			peer.NewContext(
				ctx,
				&peer.Peer{
					AuthInfo: credentials.TLSInfo{
						State: tls.ConnectionState{
							PeerCertificates: []*x509.Certificate{
								certificateValid,
							},
						},
					},
				}))
		require.NoError(t, err)
		require.Equal(t, expectedMetadata, actualMetadata)
	})
}
