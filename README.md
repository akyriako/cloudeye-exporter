# cloudeye-exporter

Prometheus cloudeye exporter for [Open Telekom Cloud](https://www.open-telekom-cloud.com/en).

## Usage
```
 ./cloudeye-exporter  -config=clouds.yml -debug 
```

The default port is `8087`, default config file location is `./clouds.yml`. If you want to enable debug mode and
have more verbose logging use the flag `-debug`. After you run the exporter you can open http://localhost:8087/metrics?services=SYS.VPC,SYS.ELB
in your browser and observe the exported metrics. 

## Help
```
Usage of ./cloudeye-exporter:
  -config string
        path to the cloud configuration file (default "./clouds.yml")
  -debug 
        provide extensive logging for debug purposes.
 
```

## Example of config file(clouds.yml)
The respective `auth_url` endpoints per region can be get found at [Identity and Access Management (IAM) endpoint list](https://developer.huaweicloud.com/en-us/endpoint).

```
global:
  prefix: "opentelekomcloud"
  scrape_batch_size: 10

auth:
  auth_url: "https://iam.eu-XX.otc.t-systems.com/v3"
  project_name: "{project_name}"
  access_key: "{access_key}"
  secret_key: "{secret_key}"
  region: "{region}"
```

## Kubernetes Installation
Consult the instructions in [README.md](deploy%2FREADME.md).