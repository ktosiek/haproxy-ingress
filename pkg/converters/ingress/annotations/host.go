/*
Copyright 2019 The HAProxy Ingress Controller Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package annotations

import (
	ingtypes "github.com/jcmoraisjr/haproxy-ingress/pkg/converters/ingress/types"
)

func (c *updater) buildHostAuthTLS(d *hostData) {
	tlsSecret := d.mapper.Get(ingtypes.HostAuthTLSSecret)
	if tlsSecret.Source == nil || tlsSecret.Value == "" {
		return
	}
	verify := d.mapper.Get(ingtypes.HostAuthTLSVerifyClient)
	if verify.Value == "off" {
		return
	}
	if cafile, err := c.cache.GetCASecretPath(tlsSecret.Value); err == nil {
		d.host.TLS.CAFilename = cafile.Filename
		d.host.TLS.CAHash = cafile.SHA1Hash
		d.host.TLS.CAVerifyOptional = verify.Value == "optional" || verify.Value == "optional_no_ca"
		d.host.TLS.CAErrorPage = d.mapper.Get(ingtypes.HostAuthTLSErrorPage).Value
	} else {
		c.logger.Error("error building TLS auth config: %v", err)
	}
}

func (c *updater) buildHostCertSigner(d *hostData) {
	signer := d.mapper.Get(ingtypes.HostCertSigner)
	if signer.Value == "" {
		return
	}
	if signer.Value != "acme" {
		c.logger.Warn("ignoring invalid cert-signer on %v: %s", signer.Source, signer.Value)
		return
	}
	if !c.acme.HasAccount() {
		c.logger.Warn("ignoring acme signer on %v due to missing endpoint or email config", signer.Source)
		return
	}
	// just warnings, ingress.syncIngress() has already added this
}

func (c *updater) buildHostSSLPassthrough(d *hostData) {
	sslpassthrough := d.mapper.Get(ingtypes.HostSSLPassthrough)
	if !sslpassthrough.Bool() {
		return
	}
	rootPath := d.host.FindPath("/")
	if rootPath == nil {
		c.logger.Warn("skipping SSL of %s: root path was not configured", sslpassthrough.Source)
		return
	}
	for _, path := range d.host.Paths {
		if path.Path != "/" {
			c.logger.Warn("ignoring path '%s' from %s: ssl-passthrough only support root path", path.Path, sslpassthrough.Source)
		}
	}
	sslpassHTTPPort := d.mapper.Get(ingtypes.HostSSLPassthroughHTTPPort)
	if sslpassHTTPPort.Source != nil {
		httpBackend := c.haproxy.FindBackend(rootPath.Backend.Namespace, rootPath.Backend.Name, sslpassHTTPPort.Value)
		if httpBackend != nil {
			d.host.HTTPPassthroughBackend = httpBackend.ID
		}
	}
	backend := c.haproxy.AcquireBackend(rootPath.Backend.Namespace, rootPath.Backend.Name, rootPath.Backend.Port)
	backend.ModeTCP = true
	d.host.SSLPassthrough = true
}

func (c *updater) buildHostTimeout(d *hostData) {
	if cfg := d.mapper.Get(ingtypes.HostTimeoutClient); cfg.Source != nil {
		d.host.Timeout.Client = c.validateTime(cfg)
	}
	if cfg := d.mapper.Get(ingtypes.HostTimeoutClientFin); cfg.Source != nil {
		d.host.Timeout.ClientFin = c.validateTime(cfg)
	}
}
