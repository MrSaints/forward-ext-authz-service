# forward-ext-authz-service

A forward authentication / authorisation (authN) implementation of [Envoy](https://www.envoyproxy.io/) [External Authorization (ext_authz)](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/ext_authz_filter), built with [Contour](https://projectcontour.io/), and [Pomerium](https://www.pomerium.com/) in mind.

_This is still under development. It works, but use at your own risk._

---

**Why do I need this?**

1. You are using an ingress controller
2. You want to delegate authN to an external Identity and Access Management (IAM) solution (e.g. Keycloak, OAuth2 Proxy, Pomerium), and have it handle the entire authN flow (with redirects)
3. The ingress controller does not directly support OAuth2, OpenID Connect (OIDC) OR any other integration with an external IAM solution you want to use (e.g. it may not implement `ext_authz`)
4. The external IAM solution you want to use supports forward authN

If the answer is "yes" to all the above, this is where `forward-ext-authz-service` comes in.

It bridges the gap between an ingress controller which _only supports_ `ext_authz`, and an external IAM solution that does not support `ext_authz`, but does support forward authN. Specifically, it was built with Contour, and Pomerium in mind.

Even if your ingress controller does support other non-Envoy authN options, you may want to consider using this as an alternative solution so that you can leverage the often simpler `ext_authz` integration instead.


## TODO

- [ ] Publish Docker image
- [ ] Create sample Kubernetes manifests
- [ ] Expand docs with diagram of authN flow
