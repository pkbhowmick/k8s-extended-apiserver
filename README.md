# k8s-extended-apiserver



## Admission Webhook
Admission webhooks are HTTP callbacks that receive admission requests and do something with them.

There are mainly two types of admission webhooks, validating admission webhook and mutating admission webhook.

### Mutating admission webhook
Mutating admission webhooks are invoked first and can modify objects sent to the API server to enforce custom defaults.

### Validating admission webhook
After all object modifications are complete, and after the incoming object is validated by the API server, validating admission webhooks are invoked and can reject requests to enforce custom policies.


## Reference:
- [How TLS and self-signed certificates work](https://www.youtube.com/watch?v=gH5X7hLAWeU)
- [DIY-k8s-extended-apiserver](https://github.com/tamalsaha/DIY-k8s-extended-apiserver)
- [kube-mutating-webhook-tutorial](https://github.com/morvencao/kube-mutating-webhook-tutorial)
- [Dynamic admission webhook k8s doc](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/)
