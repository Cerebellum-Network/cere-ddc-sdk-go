# cere-ddc-sdk-go

The Cere DDC SDK for Go.


## How to update the Protobuf schema

Clone the ddc-schemas repo:

    git submodule update --init

Checkout a particular version of the schema:

    cd ddc-schemas && git checkout storage-v0.1.2

Regenerate the code

    make protobuf

Freeze the schema version and generated code

    git add ddc-schemas model/pb
