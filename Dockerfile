# Build the manager binary
ARG GOLANG_BUILDER=golang:1.24
ARG OPERATOR_BASE_IMAGE=gcr.io/distroless/static:debug
FROM $GOLANG_BUILDER AS builder
#Arguments required by OSBS build system
ARG CACHITO_ENV_FILE=/remote-source/cachito.env
ARG REMOTE_SOURCE=.
ARG REMOTE_SOURCE_DIR=/remote-source
ARG REMOTE_SOURCE_SUBDIR=
ARG GO_BUILD_EXTRA_ARGS=
ARG DEST_ROOT=/remote-source/build
COPY $REMOTE_SOURCE $REMOTE_SOURCE_DIR
WORKDIR $REMOTE_SOURCE_DIR/$REMOTE_SOURCE_SUBDIR

RUN mkdir -p build
RUN mkdir -p ./build/usr/local/bin/
RUN pwd


# Copy Go modules manifests
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY cmd/ cmd/
COPY api/ api/


RUN if [ ! -f $CACHITO_ENV_FILE ]; then go mod download ; fi

# Build manager
RUN if [ -f $CACHITO_ENV_FILE ] ; then source $CACHITO_ENV_FILE ; fi ; CGO_ENABLED=0  GO111MODULE=on go build ${GO_BUILD_EXTRA_ARGS} -a -o ${DEST_ROOT}/manager cmd/main.go



# RUN cp -r templates ${DEST_ROOT}/templates

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM $OPERATOR_BASE_IMAGE
ARG USER_ID=nonroot:nonroot
ARG IMAGE_COMPONENT="ciscoaci-aim-operator-controller"
ARG IMAGE_NAME="ciscoaci-aim-operator"
ARG IMAGE_VERSION="1.0.0"
ARG IMAGE_SUMMARY="Cisco Aci Aim Operator"
ARG IMAGE_DESC="This image includes the ciscoaci-aim-operator"
ARG IMAGE_TAGS="cn-openstack openstack"


# Install operator binary to WORKDIR
COPY --from=builder ./remote-source/build/manager .

USER $USER_ID
ENV PATH="/:${PATH}"

ENTRYPOINT ["/manager"]
