package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

func StartClient(url_, heads, requestBody string, proxy string, meth string, dka bool, responseChan chan *Response, tc int) {
	var tr *http.Transport
	u, err := url.Parse(url_)

	if err == nil && u.Scheme == "https" {
		var tlsConfig *tls.Config
		if *insecure {
			tlsConfig = &tls.Config{
				InsecureSkipVerify: true,
			}
		} else {
			// Load client cert
			cert, err := tls.LoadX509KeyPair(*certFile, *keyFile)
			if err != nil {
				log.Fatal(err)
			}

			// Load CA cert
			caCert, err := os.ReadFile(*caFile)
			if err != nil {
				log.Fatal(err)
			}
			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(caCert)

			// Setup HTTPS client
			tlsConfig = &tls.Config{
				Certificates: []tls.Certificate{cert},
				RootCAs:      caCertPool,
			}
		}

		tr = &http.Transport{TLSClientConfig: tlsConfig, DisableKeepAlives: dka}
	} else {
		tr = &http.Transport{DisableKeepAlives: dka}
	}

	var proxyURL *url.URL
	if proxy != "" {
		proxyURL, err = url.Parse(proxy)
		if err != nil {
			fmt.Printf("Error parsing proxy URL: %v\n", err)
			os.Exit(1)
		}

		tr.Proxy = http.ProxyURL(proxyURL)
	}

	timer := NewTimer()

	hdrs, _ := buildHeaders(heads)

	for {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		if *random {
			q := u.Query()
			q.Set("random", fmt.Sprint(r.Int()))
			u.RawQuery = q.Encode()
		}

		requestBodyReader := strings.NewReader(requestBody)
		req, _ := http.NewRequest(meth, u.String(), requestBodyReader)

		for key, vals := range hdrs {
			for _, val := range vals {
				req.Header.Set(key, val)
			}
		}

		timer.Reset()

		resp, err := tr.RoundTrip(req)
		defer resp.Body.Close()

		respObj := &Response{}

		if err != nil {
			respObj.Error = true
		} else {
			if resp.ContentLength < 0 { // -1 if the length is unknown
				data, err := io.ReadAll(resp.Body)
				if err == nil {
					respObj.Size = int64(len(data))
				}
			} else {
				respObj.Size = resp.ContentLength
				if *respContains != "" {
					data, err := io.ReadAll(resp.Body)
					if err == nil {
						respObj.Body = string(data)
					}
				}
			}
			respObj.StatusCode = resp.StatusCode
		}

		respObj.Duration = timer.Duration()

		if len(responseChan) >= tc {
			break
		}
		responseChan <- respObj
	}
}

// buildHeaders build the HTTP Request headers from the parsed flag -H or
// from the default header set.
// The headers are "set" (not added), thus same key values get replaced.
// Note: if a key has no value, it is not added into the Headers, by original
// package design.
func buildHeaders(heads string) (http.Header, error) {

	heads = strings.Replace(heads, `\n`, "\n", -1)
	h := http.Header{}

	sets := strings.Split(heads, "\n")
	for i := range sets {
		split := strings.SplitN(sets[i], ":", 2)
		if len(split) == 2 {
			h.Set(split[0], split[1])
		}
	}

	return h, nil
}
