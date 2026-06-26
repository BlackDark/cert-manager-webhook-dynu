# Dynu Solver Test Data

Before running tests, copy the file `examples/dynu-api-key.yaml` to this directory, and substitute **_<API-Key-(Base64-encoded)>_** with your base64-encoded Dynu API key.

Run the test with the following, setting **TEST_ZONE_NAME** to one of your Dynu DNS zones.

```bash
$ TEST_ZONE_NAME=example.com. make test
```
