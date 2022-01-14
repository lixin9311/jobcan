package jobcan

import (
	"net/http"
	"net/url"
	"time"

	cookiejar "github.com/juju/persistent-cookiejar"
)

var _ http.CookieJar = (*Jar)(nil)

type Jar struct {
	*cookiejar.Jar
}

func NewJar(o *cookiejar.Options) *Jar {
	jar, _ := cookiejar.New(o)
	return &Jar{
		Jar: jar,
	}
}

func (j *Jar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	for _, c := range cookies {
		if c.MaxAge == 0 {
			// hack
			c.MaxAge = 1800
		}
	}
	j.Jar.SetCookies(u, cookies)
}

func (j *Jar) CheckSession() (expired bool) {
	future := time.Now().Add(time.Minute)
	cookies := j.AllCookies()
	if len(cookies) == 0 {
		return true
	}
	for _, c := range cookies {
		if c.Expires.Before(future) {
			return true
		}
	}
	return false
}
