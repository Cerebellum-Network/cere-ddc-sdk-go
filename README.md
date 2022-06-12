# cere-ddc-sdk-go

The Cere DDC SDK for Go.


## How to update the Protobuf schema

Clone the ddc-schemas repo:

    git submodule update --init

Checkout a particular version of the schema:

    cd ddc-schemas && git checkout storage-vX.X.X

Regenerate the code through a Docker image

    make protobuf

    … or through the `protoc` command …

    make protoc

Freeze the schema version and generated code

    git add ddc-schemas model/pb

