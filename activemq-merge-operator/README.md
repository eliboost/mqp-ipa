# activemq-merge-operator
This project is based on the gitlab-controller-runtime package.

## Entwicklung
### Mocks generieren
Mocks können wie folgt generiert werden:

* Mockery installieren:

```
go install github.com/vektra/mockery/v2@latest
```

* Mit dem command `mockery` die Mocks generieren lassen und ins Repo einchecken. Die Konfiguration dazu befindet sich im file `.mockery.yaml`
