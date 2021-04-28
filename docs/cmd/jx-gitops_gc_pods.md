## jx-gitops gc pods

garbage collection for pods

***Aliases**: pod*

### Usage

```
jx-gitops gc pods
```

### Synopsis

Garbage collect old Pods that have completed or failed

### Examples

  # garbage collect old pods of the default age
  jx gitops gc pods
  
  # garbage collect pods older than 10 minutes
  jx gitops gc pods -a 10m

### Options

```
  -a, --age duration       The minimum age of pods to garbage collect. Any newer pods will be kept (default 1h0m0s)
  -h, --help               help for pods
  -n, --namespace string   The namespace to look for the pods. Defaults to the current namespace
  -s, --selector string    The selector to use to filter the pods
```

### SEE ALSO

* [jx-gitops gc](jx-gitops_gc.md)	 - Commands for garbage collecting resources

###### Auto generated by spf13/cobra on 28-Apr-2021