package serviceprovider

import (
	"net/http"
	"time"

	"github.com/pkg/errors"

	"github.com/aws/aws-sdk-go/aws"
	aws_creds "github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/cloudflare/cfssl/log"
	"github.com/jpillora/backoff"
	"github.com/wjessop/aws-utils.go/credentials"
)

const (
	servicePoolSize = 10
)

var (
	awsOperationRetryCount = 50
)

// S3ServiceProvider wraps logic for providing a s3 service object and keeping it fresh
type S3ServiceProvider struct {
	cfg         *aws.Config
	servicePool chan *s3.S3
	bucket      string
}

// NewS3ServiceProvider creates a new S3ServiceProvider. Credentials can be an aws credentials.Credentials
// object directly (*"github.com/wjessop/aws-utils.go/credentials".Credentials) or something that
// looks like it
func NewS3ServiceProvider(region, bucketName string, client *http.Client, creds credentials.Credentials) *S3ServiceProvider {
	svp := &S3ServiceProvider{
		cfg:         aws.NewConfig().WithHTTPClient(client).WithRegion(region).WithCredentials(creds.(*aws_creds.Credentials)), //.WithLogLevel(aws.LogDebugWithRequestRetries | aws.LogDebugWithRequestErrors),
		servicePool: make(chan *s3.S3, servicePoolSize),
		bucket:      bucketName,
	}
	return svp
}

func (s *S3ServiceProvider) getS3Service() *s3.S3 {
	b := &backoff.Backoff{
		Min:    100 * time.Millisecond,
		Max:    5 * time.Minute,
		Factor: 2,
		Jitter: true,
	}

	for i := 0; i < awsOperationRetryCount; i++ {
		sess, err := session.NewSession()

		if err != nil {
			if i < awsOperationRetryCount {
				sleep := b.Duration()
				log.Warningf("Got error creating AWS service. Will sleep for %v seconds: %v", sleep, err)
				time.Sleep(sleep)
				continue
			} else {
				log.Fatalf("Unable to get valid response from AWS after %d tries: %v", awsOperationRetryCount, err)
			}
		}

		svc := s3.New(sess, s.cfg)

		if svc.Config.Credentials.IsExpired() {
			if i < awsOperationRetryCount {
				log.Warningf("Got error creating AWS service, will retry")
				continue
			} else {
				log.Fatal("AWS creds expired or invalid", errors.New("Credentials from environment are invalid or expired"))
			}
		}

		return svc
	}

	// This return is unreachable
	return nil
}

// GetS3Service gets the currently created s3 service
func (s *S3ServiceProvider) GetS3Service() *s3.S3 {
	select {
	case svc := <-s.servicePool:
		log.Debug("Providing S3 Service from the pool")
		return svc
	default:
		log.Debug("Generating new S3 Service")
		return s.getS3Service()
	}
}

// ReturnS3Service accepts an S3 service and either adds it to the pool or dumps it
func (s *S3ServiceProvider) ReturnS3Service(svc *s3.S3) {
	select {
	case s.servicePool <- svc:
		log.Debug("Returning S3 Service to the pool")
	default:
		log.Debug("Discarding S3 Service as pool is full")
	}
	return

}
