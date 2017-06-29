# Helm steer examples

This directory contains a catalog of examles on how to run, upgrade and manage
charts in a cluster using Helm steer

All plans can be run with the following command

```
$ helm steer -d -v <filename>
```

Name                | File                          | Description
--------------------|-------------------------------|------------
Single chart        | [single_chart.yaml]() | Installs a single chart in a single namespace
Chart upgrade       | [single_chart_upgrade.yaml]() | Upgrade a chart to a new version
Multiple namespaces | [multiple_namespaces.yaml]() | Install multiple charts in multiple namespaces
Dependencies        | [dependencies.yaml]() | Installs charts in the proper order according to explicit dependencies
Error management    | [error.yaml]() | Performs the recovery operation when a chart install failed



