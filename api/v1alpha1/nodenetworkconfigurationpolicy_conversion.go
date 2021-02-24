package v1alpha1

/*
Implementing the hub method is pretty easy -- we just have to add an empty
method called `Hub()` to serve as a
[marker](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/conversion?tab=doc#Hub).
We could also just put this inline in our `nodenetworkconfigurationpolicy_types.go` file.
*/

// Hub marks this type as a conversion hub.
func (*NodeNetworkConfigurationPolicy) Hub() {}
