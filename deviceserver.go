package deviceserver

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/square/go-jose"
	l "gitlab.flowcloud.systems/creator-ops/logger"
)

type Config struct {
	BaseUrl       string
	PSK           string
	SkipTLSVerify bool
	Log           l.Logger
}

type Client struct {
	baseUrl      string
	signer       JwtSigner
	client       *http.Client
	cache        *cache.Cache
	authGetJson  map[string]string
	authPostJson map[string]string
	log          l.Logger
}

func Create(config *Config) (*Client, error) {
	ds := Client{
		baseUrl: config.BaseUrl,
		cache:   cache.New(120*time.Second, 30*time.Second),
		log:     config.Log,
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: config.SkipTLSVerify},
	}

	ds.client = &http.Client{
		Transport: tr,
	}

	ds.authGetJson = map[string]string{
		"Accept": "application/json",
	}

	ds.authPostJson = map[string]string{
		"Accept":       "application/json",
		"Content-Type": "application/json",
	}

	err := ds.signer.Init(jose.HS256, []byte(config.PSK))
	if err != nil {
		return nil, err
	}

	return &ds, nil
}

func (d *Client) Authorize(req *http.Request) error {

	// the lifetime should be shorter, but think I'm hitting some timezone issues at the moment
	orgClaim := OrgClaim{
		OrgID: 0,
		Exp:   time.Now().Add(60 * time.Minute).Unix(),
	}

	serialized, err := d.signer.MarshallSignSerialize(orgClaim)
	if err != nil {
		return err
	}
	//fmt.Println(serialized)

	req.Header.Set("Authorization", "Bearer "+serialized)
	return nil
}

func (d *Client) Get(url string, headers map[string]string, result interface{}) error {
	// cached, ok := d.cache.Get(url)
	// if ok {
	// 	err := interfacetools.CopyOut(cached, result)
	// 	fmt.Println(cached)
	// 	fmt.Println(result)
	// 	return err
	// }

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		d.log.Request(req, "NewRequest(): %s", err.Error())
		return err
	}

	for n, v := range headers {
		req.Header.Set(n, v)
	}

	err = d.Authorize(req)
	if err != nil {
		d.log.Request(req, "Authorize(): %s", err.Error())
		return err
	}

	resp, err := d.client.Do(req)
	if err != nil {
		d.log.Request(req, "Do(): %s", err.Error())
		return err
	}
	if resp.StatusCode > 400 {
		d.log.Request(req, "status: %s(%d)", resp.Status, resp.StatusCode)
		return fmt.Errorf("http status: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		d.log.Request(req, "ReadAll(): %s", err.Error())
		return err
	}
	resp.Body.Close()

	err = json.Unmarshal(body, result)
	if err != nil {
		d.log.Request(req, "Unmarshal(): %s", err.Error())
		return err
	}

	d.log.Request(req, "Get(): OK")
	d.cache.Set(url, result, cache.DefaultExpiration)
	return nil
}

func (d *Client) Post(url string, headers map[string]string, postbody io.Reader, result interface{}) error {

	req, err := http.NewRequest("POST", url, postbody)
	if err != nil {
		d.log.Request(req, "NewRequest(): %s", err.Error())
		return err
	}

	for n, v := range headers {
		req.Header.Set(n, v)
	}

	err = d.Authorize(req)
	if err != nil {
		d.log.Request(req, "Authorize(): %s", err.Error())
		return err
	}

	resp, err := d.client.Do(req)
	if err != nil {
		d.log.Request(req, "Do(): %s", err.Error())
		return err
	}
	if resp.StatusCode > 400 {
		d.log.Request(req, "status: %s(%d)", resp.Status, resp.StatusCode)
		return fmt.Errorf("http status: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		d.log.Request(req, "ReadAll(): %s", err.Error())
		return err
	}
	resp.Body.Close()

	err = json.Unmarshal(body, result)
	if err != nil {
		d.log.Request(req, "Unmarshal(): %s", err.Error())
		return err
	}

	d.log.Request(req, "Post(): OK")
	return nil
}

func (d *Client) CreateAccessKey(name string) (*AccessKey, error) {
	var entry EntryPoint

	err := d.Get(d.baseUrl, d.authGetJson, &entry)
	if err != nil {
		d.log.Error("Get(): %s", err.Error())
		return nil, err
	}

	accesskeys, err := entry.Links.GetLink("accesskeys")
	if err != nil {
		d.log.Error("GetLink(): %s", err.Error())
		return nil, err
	}

	var key AccessKey
	err = d.Post(accesskeys.Href,
		d.authPostJson,
		bytes.NewBuffer([]byte(fmt.Sprintf(`{"Name":"%s"}`, name))),
		&key)

	return &key, nil
}
