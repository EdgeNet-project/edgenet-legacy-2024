# EdgeNet

## RBAC

### Cluster roles and role bindings

#### Conventions

Cluster role file name format is as following:
```
<verb>-<resource>.yml
```

Cluster role metadata name format:

```
edgenet:<verb>:<resource>
```

Cluster role binding file name format:

```
<user>-<resource>.yml
```

and metadata name format:
```
edgenet:<user>:<resource>
```

#### User/Authority registration
For user registration to work we need the anonymous user to have access to the following verbs/resources:

- get/list authorities
- create authorityrequest
- create userregistrationrequests
- patch emailverification

