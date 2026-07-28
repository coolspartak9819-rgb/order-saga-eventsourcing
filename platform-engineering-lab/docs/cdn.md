# CDN and Cache Strategy

The origin nginx layer sets `Cache-Control` for public API content and exposes
`X-Cache-Status` for diagnostics. A production CDN can sit in front of nginx
using the same cache headers.

Recommended policy:

- cache only GET and HEAD requests;
- do not cache authenticated or personalized responses;
- use short TTLs for frequently changing content;
- use versioned asset URLs for static files;
- purge by surrogate key or path after a content release;
- keep stale content available during a short origin outage.

For Yandex Cloud CDN or Selectel CDN, the origin should be the public nginx
endpoint. TLS terminates at the CDN or an edge reverse proxy, and the origin
still accepts only the required network traffic.
