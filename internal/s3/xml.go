package s3

import (
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"time"
)


const S3Namespace = "http://s3.amazonaws.com/doc/2006-03-01/"

// Owner represents the owner tag in S3 XML responses.
type Owner struct {
	ID          string `xml:"ID"`
	DisplayName string `xml:"DisplayName"`
}

// Bucket represents a single bucket in ListAllMyBucketsResult.
type Bucket struct {
	Name         string `xml:"Name"`
	CreationDate string `xml:"CreationDate"`
}

// ListAllMyBucketsResult represents GET / XML output.
type ListAllMyBucketsResult struct {
	XMLName xml.Name `xml:"ListAllMyBucketsResult"`
	Xmlns   string   `xml:"xmlns,attr"`
	Owner   Owner    `xml:"Owner"`
	Buckets struct {
		Bucket []Bucket `xml:"Bucket"`
	} `xml:"Buckets"`
}

// ObjectContent represents an object entry in ListBucketResult (ListObjectsV2).
type ObjectContent struct {
	Key          string `xml:"Key"`
	LastModified string `xml:"LastModified"`
	ETag         string `xml:"ETag"`
	Size         int64  `xml:"Size"`
	StorageClass string `xml:"StorageClass"`
}

// CommonPrefix represents virtual folder prefixes in ListBucketResult.
type CommonPrefix struct {
	Prefix string `xml:"Prefix"`
}

// ListBucketResult represents GET /{bucket}?list-type=2 XML output.
type ListBucketResult struct {
	XMLName               xml.Name        `xml:"ListBucketResult"`
	Xmlns                 string          `xml:"xmlns,attr"`
	Name                  string          `xml:"Name"`
	Prefix                string          `xml:"Prefix"`
	KeyCount              int             `xml:"KeyCount"`
	MaxKeys               int             `xml:"MaxKeys"`
	Delimiter             string          `xml:"Delimiter,omitempty"`
	IsTruncated           bool            `xml:"IsTruncated"`
	Contents              []ObjectContent `xml:"Contents"`
	CommonPrefixes        []CommonPrefix  `xml:"CommonPrefixes,omitempty"`
	NextContinuationToken string          `xml:"NextContinuationToken,omitempty"`
	ContinuationToken     string          `xml:"ContinuationToken,omitempty"`
	StartAfter            string          `xml:"StartAfter,omitempty"`
}

// InitiateMultipartUploadResult represents POST /{bucket}/{key}?uploads XML output.
type InitiateMultipartUploadResult struct {
	XMLName  xml.Name `xml:"InitiateMultipartUploadResult"`
	Xmlns    string   `xml:"xmlns,attr"`
	Bucket   string   `xml:"Bucket"`
	Key      string   `xml:"Key"`
	UploadId string   `xml:"UploadId"`
}

// CompleteMultipartUpload represents input body for completing a multipart upload.
type CompleteMultipartUpload struct {
	XMLName xml.Name `xml:"CompleteMultipartUpload"`
	Parts   []struct {
		PartNumber int    `xml:"PartNumber"`
		ETag       string `xml:"ETag"`
	} `xml:"Part"`
}

// CompleteMultipartUploadResult represents POST /{bucket}/{key}?uploadId=X XML output.
type CompleteMultipartUploadResult struct {
	XMLName  xml.Name `xml:"CompleteMultipartUploadResult"`
	Xmlns    string   `xml:"xmlns,attr"`
	Location string   `xml:"Location"`
	Bucket   string   `xml:"Bucket"`
	Key      string   `xml:"Key"`
	ETag     string   `xml:"ETag"`
}

// ErrorResult represents standard S3 error responses in XML format.
type ErrorResult struct {
	XMLName   xml.Name `xml:"Error"`
	Code      string   `xml:"Code"`
	Message   string   `xml:"Message"`
	Resource  string   `xml:"Resource"`
	RequestId string   `xml:"RequestId"`
}

// FormatISO8601 formats time in S3 ISO8601 date format.
func FormatISO8601(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05.000Z")
}

// WriteXML renders an XML payload to the HTTP response with appropriate status code and headers.
func WriteXML(w http.ResponseWriter, statusCode int, v interface{}) {
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(statusCode)
	w.Write([]byte(xml.Header))
	_ = xml.NewEncoder(w).Encode(v)
}

// WriteError renders a standard S3 XML error payload with global error message sanitization (finding #25).
func WriteError(w http.ResponseWriter, statusCode int, code, message, resource string) {

	reqID := fmt.Sprintf("%d", time.Now().UnixNano())
	cleanMsg := message
	if statusCode >= 500 || code == "InternalError" {
		log.Printf("[S3 Error %s] %s (resource: %s, reqID: %s)", code, message, resource, reqID)
		cleanMsg = "We encountered an internal error. Please try again."
	}
	errRes := ErrorResult{
		Code:      code,
		Message:   cleanMsg,
		Resource:  resource,
		RequestId: reqID,
	}
	WriteXML(w, statusCode, errRes)
}

