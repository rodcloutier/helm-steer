


charts:
    # use development versions, too. Equivalent to version '>0.0.0-a'. If --version is set, this is ignored.
  - devel: false
    # simulate an install
    dry-run: false
    # location of public keys used for verification (default "/Users/rod/.gnupg/pubring.gpg")
    keyring: ""
    # namespace to install the release into
    namespace: ""
    # prevent hooks from running during install
    no-hooks: false
    # set values on the command line (can specify multiple or separate values with commas: key1=val1,key2=val2)
    set:
    - ""
    # time in seconds to wait for any individual kubernetes operation (like Jobs for hooks) (default 300)
    timeout: 300
    # enable TLS for request
    tls: false
    # path to TLS CA certificate file (default "$HELM_HOME/ca.pem")
    tls-ca-cert: ""
    # path to TLS certificate file (default "$HELM_HOME/cert.pem")
    tls-cert: ""
    # path to TLS key file (default "$HELM_HOME/key.pem")
    tls-key: ""
    # enable TLS for request and verify remote
    tls-verify: false
    # specify values in a YAML file (can specify multiple) (default [])
    values:
    - ""
    # verify the package before installing it
    verify: true
    # specify the exact chart version to install. If this is not specified, the latest version is installed
    version: ""
    # if set, will wait until all Pods, PVCs, Services, and minimum number of Pods of a Deployment are in a ready state before marking the release as successful. It will wait for as long as --timeout
    wait: true

    # INSTALL specific ------------------------------------------------
    # release name. If unspecified, it will autogenerate one for you
    # (rod) Must be converted to argument to upgrade
    name: ""
    # specify template used to name the release
    name-template: ""
    # re-use the given name, even if that name is already used. This is unsafe in production
    replace: true


    # UPGRADE specific -----------------------------------------------
    # if a release by this name doesn't already exist, run an install
    # install: false
    # performs pods restart for the resource if applicable
    recreate-pods: false
    # when upgrading, reset the values to the ones built into the chart
    reset-values: false
    # when upgrading, reuse the last release's values, and merge in any
    reuse-values: false

