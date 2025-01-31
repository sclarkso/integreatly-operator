include ./make/*.mk

ORG ?= integreatly

REG=quay.io
SHELL=/bin/bash

PKG=github.com/integr8ly/integreatly-operator
TEST_DIRS?=$(shell sh -c "find $(TOP_SRC_DIRS) -name \\*_test.go -exec dirname {} \\; | sort | uniq")
TEST_POD_NAME=integreatly-operator-test
COMPILE_TARGET=./tmp/_output/bin/$(PROJECT)
OPERATOR_SDK_VERSION=1.21.0
AUTH_TOKEN=$(shell curl -sH "Content-Type: application/json" -XPOST https://quay.io/cnr/api/v1/users/login -d '{"user": {"username": "$(QUAY_USERNAME)", "password": "$(QUAY_PASSWORD)"}}' | jq -r '.token')
TEMPLATE_PATH="$(shell pwd)/templates/monitoring"
IN_PROW ?= "false"
# DEV_QUOTA value is the default QUOTA when install locally and is per 100,000
# acceptable values are
# if 10 then 1M
# if 50 then 5M
# if 100 then 10M
# if 200 then 20M
# if 500 then 50M
# if 1 then 100k
DEV_QUOTA ?= "1"
SMTP_USER  ?= ''
SMTP_ADDRESS ?= ''
SMTP_PASS ?= ''
SMTP_PORT ?= ''
SMTP_FROM ?= ''
TYPE_OF_MANIFEST ?= master

CONTAINER_ENGINE ?= docker
TEST_RESULTS_DIR ?= test-results
TEMP_SERVICEACCOUNT_NAME=rhmi-operator
SANDBOX_NAMESPACE ?= sandbox-rhoam-operator

# These tags are modified by the prepare-release script.
RHMI_TAG ?= 2.9.0
RHOAM_TAG ?= 1.24.0

export SKIP_FLAKES := true

# If openapi-gen is available on the path, use that; otherwise use it through
# "go run" (slower)
ifneq (, $(shell which openapi-gen 2> /dev/null))
	OPENAPI_GEN ?= openapi-gen
else
	OPENAPI_GEN ?= go run k8s.io/kube-openapi/cmd/openapi-gen
endif

# If the _correct_ version of operator-sdk is on the path, use that (faster);
# otherwise use it through "go run" (slower but will always work and will use correct version)
ifeq ($(shell operator-sdk version 2> /dev/null | sed -e 's/", .*/"/' -e 's/.* //'), "v$(OPERATOR_SDK_VERSION)")
	OPERATOR_SDK ?= operator-sdk
else
	OPERATOR_SDK ?= go run github.com/operator-framework/operator-sdk/cmd/operator-sdk
endif

# Set sed -i as it's different for mac vs gnu
ifeq ($(shell uname -s | tr A-Z a-z), darwin)
	SED_INLINE ?= sed -i ''
else
 	SED_INLINE ?= sed -i
endif

export SELF_SIGNED_CERTS   ?= true
# Setting the INSTALLATION_TYPE to managed-api will configure the values required for RHOAM installs
export INSTALLATION_TYPE   ?= managed

export ALERT_SMTP_FROM ?= noreply-alert@devshift.org
export USE_CLUSTER_STORAGE ?= true
export OPERATORS_IN_PRODUCT_NAMESPACE ?= false # e2e tests and createInstallationCR() need to be updated when default is changed
export DELOREAN_PULL_SECRET_NAME ?= integreatly-delorean-pull-secret
export ALERTING_EMAIL_ADDRESS ?= noreply-test@rhmi-redhat.com
export BU_ALERTING_EMAIL_ADDRESS ?= noreply-test@rhmi-redhat.com

ifeq ($(shell test -e envs/$(INSTALLATION_TYPE).env && echo -n yes),yes)
	include envs/$(INSTALLATION_TYPE).env
endif

define wait_command
	@echo Waiting for $(2) for $(3)...
	@time timeout --foreground $(3) bash -c "until $(1); do echo $(2) not ready yet, trying again in $(4)s...; sleep $(4); done"
	@echo $(2) ready!
endef

# Options for 'bundle-build'
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# BUNDLE_GEN_FLAGS are the flags passed to the operator-sdk generate bundle command
BUNDLE_GEN_FLAGS ?= -q --overwrite --version $(TAG) $(BUNDLE_METADATA_OPTS)

# USE_IMAGE_DIGESTS defines if images are resolved via tags or digests
# You can enable this value if you would like to use SHA Based Digests
# To enable set flag to true
USE_IMAGE_DIGESTS ?= false
ifeq ($(USE_IMAGE_DIGESTS), true)
    BUNDLE_GEN_FLAGS += --use-image-digests
endif

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

.PHONY: setup/moq
setup/moq:
	GO111MODULE=off go get github.com/matryer/moq

.PHONY: setup/service_account/oc_login
setup/service_account/oc_login:
	@oc login --token=$(shell sh -c "oc serviceaccounts get-token rhmi-operator -n ${NAMESPACE}") --server=$(shell sh -c "oc whoami --show-server") --kubeconfig=TMP_SA_KUBECONFIG --insecure-skip-tls-verify=true

.PHONY: setup/service_account
setup/service_account: kustomize
	@-oc new-project $(NAMESPACE)
	@oc project $(NAMESPACE)
	@-oc create -f config/rbac/service_account.yaml -n $(NAMESPACE)
	@$(KUSTOMIZE) build config/rbac-$(INSTALLATION_SHORTHAND) | oc replace --force -f -
	$(MAKE) setup/service_account/oc_login

.PHONY: setup/git/hooks
setup/git/hooks:
	git config core.hooksPath .githooks

.PHONY: install/sandboxrhoam/operator
install/sandboxrhoam/operator:
	@-oc new-project $(SANDBOX_NAMESPACE)
	@-oc process -p RHOAM_NAMESPACE=$(SANDBOX_NAMESPACE) -f config/developer-sandbox/sandbox-operator-template.yml | oc create -f - -n $(SANDBOX_NAMESPACE)

.PHONY: install/sandboxrhoam/config
install/sandboxrhoam/config:
	@-oc process -p RHOAM_OPERATOR_NAMESPACE=$(SANDBOX_NAMESPACE) -f config/developer-sandbox/sandbox-config-template.yml | oc create -f - -n $(SANDBOX_NAMESPACE)
	@oc label namespace $(SANDBOX_NAMESPACE) monitoring-key=middleware --overwrite
	@-oc process -f config/developer-sandbox/sandbox-rhoam-quickstart.yml | oc create -f -

.PHONY: code/run
code/run: code/gen cluster/prepare/smtp cluster/prepare/dms cluster/prepare/pagerduty setup/service_account
	@KUBECONFIG=TMP_SA_KUBECONFIG WATCH_NAMESPACE=$(NAMESPACE) QUOTA=$(DEV_QUOTA) go run ./main.go

.PHONY: code/rerun
code/rerun: setup/service_account
	@KUBECONFIG=TMP_SA_KUBECONFIG WATCH_NAMESPACE=$(NAMESPACE) go run ./main.go

.PHONY: code/run/service_account
code/run/service_account: code/run

.PHONY: code/run/delorean
code/run/delorean: cluster/cleanup cluster/prepare cluster/prepare/local deploy/integreatly-rhmi-cr.yml code/run/service_account

.PHONY: code/compile
code/compile: code/gen
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o=$(COMPILE_TARGET) .

pkg/apis/integreatly/v1alpha1/zz_generated.openapi.go: apis/v1alpha1/rhmi_types.go
	$(OPENAPI_GEN) --logtostderr=true -o "" \
		-i ./apis/v1alpha1/ \
		-p ./apis/v1alpha1/ \
		-O zz_generated.openapi \
		-h ./hack/boilerplate.go.txt \
		-r "-"

apis/integreatly/v1alpha1/zz_generated.deepcopy.go: controller-gen apis/v1alpha1/rhmi_types.go
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: code/gen
code/gen: setup/moq apis/integreatly/v1alpha1/zz_generated.deepcopy.go
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases
	@go generate ./...
	mv ./config/crd/bases/integreatly.org_apimanagementtenants.yaml ./config/crd-sandbox/bases

.PHONY: code/check
code/check:
	@diff -u <(echo -n) <(gofmt -d `find . -type f -name '*.go' -not -path "./vendor/*"`)
	golint ./pkg/... | grep -v  "comment on" | grep -v "or be unexported"
	go vet ./...


.PHONY: code/fix
code/fix:
	@gofmt -w `find . -type f -name '*.go' -not -path "./vendor/*"`

.PHONY: image/build
image/build: code/gen
	echo "build image $(OPERATOR_IMAGE)"
	docker build -t ${OPERATOR_IMAGE} .

.PHONY: image/push
image/push:
	echo "push image $(OPERATOR_IMAGE)"
	docker push $(OPERATOR_IMAGE)

.PHONY: image/build/push
image/build/push: image/build image/push


############ E2E TEST COMMANDS ############
.PHONY: test/e2e/prow
test/e2e/prow: export component := integreatly-operator
test/e2e/prow: export OPERATOR_IMAGE := ${IMAGE_FORMAT}
test/e2e/prow: export INSTALLATION_TYPE := managed
test/e2e/prow: export SKIP_FLAKES := $(SKIP_FLAKES)
test/e2e/prow: export WATCH_NAMESPACE := redhat-rhmi-operator
test/e2e/prow: export NAMESPACE_PREFIX := redhat-rhmi-
test/e2e/prow: export INSTALLATION_PREFIX := redhat-rhmi
test/e2e/prow: export INSTALLATION_NAME := rhmi
test/e2e/prow: export INSTALLATION_SHORTHAND := rhmi
test/e2e/prow: IN_PROW = "true"
test/e2e/prow: test/e2e

.PHONY: test/e2e/rhoam/prow
test/e2e/rhoam/prow: export component := integreatly-operator
test/e2e/rhoam/prow: export OPERATOR_IMAGE := ${IMAGE_FORMAT}
test/e2e/rhoam/prow: export INSTALLATION_TYPE := managed-api
test/e2e/rhoam/prow: export SKIP_FLAKES := $(SKIP_FLAKES)
test/e2e/rhoam/prow: export WATCH_NAMESPACE := redhat-rhoam-operator
test/e2e/rhoam/prow: export NAMESPACE_PREFIX := redhat-rhoam-
test/e2e/rhoam/prow: export INSTALLATION_PREFIX := redhat-rhoam
test/e2e/rhoam/prow: export INSTALLATION_NAME := rhoam
test/e2e/rhoam/prow: export INSTALLATION_SHORTHAND := rhoam
test/e2e/rhoam/prow: IN_PROW = "true"
test/e2e/rhoam/prow: test/e2e

.PHONY: test/e2e/multitenant-rhoam/prow
test/e2e/multitenant-rhoam/prow: export component := integreatly-operator
test/e2e/multitenant-rhoam/prow: export OPERATOR_IMAGE := ${IMAGE_FORMAT}
test/e2e/multitenant-rhoam/prow: export INSTALLATION_TYPE := multitenant-managed-api
test/e2e/multitenant-rhoam/prow: export SKIP_FLAKES := $(SKIP_FLAKES)
test/e2e/multitenant-rhoam/prow: export WATCH_NAMESPACE := sandbox-rhoam-operator
test/e2e/multitenant-rhoam/prow: export NAMESPACE_PREFIX := sandbox-rhoam-
test/e2e/multitenant-rhoam/prow: export INSTALLATION_PREFIX := sandbox-rhoam
test/e2e/multitenant-rhoam/prow: export INSTALLATION_NAME := rhoam
test/e2e/multitenant-rhoam/prow: export INSTALLATION_SHORTHAND := sandbox
test/e2e/multitenant-rhoam/prow: IN_PROW = "true"
test/e2e/multitenant-rhoam/prow: test/e2e

.PHONY: test/e2e
test/e2e: export SURF_DEBUG_HEADERS=1
test/e2e: cluster/deploy
	go clean -testcache && go test -v ./test/e2e -timeout=120m -ginkgo.v

.PHONY: test/e2e/single
test/e2e/single: export WATCH_NAMESPACE := $(NAMESPACE)
test/e2e/single: 
	go clean -testcache && go test ./test/functional -ginkgo.focus="$(TEST).*" -test.v -ginkgo.v -ginkgo.progress -timeout=80m

.PHONY: test/functional
test/functional: export WATCH_NAMESPACE := $(NAMESPACE)
test/functional:
	# Run the functional tests against an existing cluster. Make sure you have logged in to the cluster.
	go clean -testcache && go test -v ./test/functional -timeout=80m

.PHONY: test/osde2e
test/osde2e: export WATCH_NAMESPACE := $(NAMESPACE)
test/osde2e: export SKIP_FLAKES := $(SKIP_FLAKES)
test/osde2e:
	# Run the osde2e tests against an existing cluster. Make sure you have logged in to the cluster.
	go clean -testcache && go test ./test/osde2e -test.v -ginkgo.v -ginkgo.progress -timeout=120m

############ E2E TEST COMMANDS ############


.PHONY: test/products/local
test/products/local: export WATCH_NAMESPACE := $(NAMESPACE)
test/products/local:
	# Running the products tests against an existing cluster inside a container. Make sure you have logged in to the cluster.
	# Using 'test-containers.yaml' as config and 'test-results' as output dir
	mkdir -p "test-results"
	$(CONTAINER_ENGINE) pull quay.io/integreatly/delorean-cli:master
	$(CONTAINER_ENGINE) run --rm -e KUBECONFIG=/kube.config -v "${HOME}/.kube/config":/kube.config:z -v $(shell pwd)/test-containers.yaml:/test-containers.yaml -v $(shell pwd)/test-results:/test-results quay.io/integreatly/delorean-cli:master delorean pipeline product-tests --test-config ./test-containers.yaml --output /test-results --namespace test-products

.PHONY: test/products
test/products: export WATCH_NAMESPACE := $(NAMESPACE)
test/products:
	# Running the products tests against an existing cluster. Make sure you have logged in to the cluster.
	# Using "test-containers.yaml" as config and $(TEST_RESULTS_DIR) as output dir
	mkdir -p $(TEST_RESULTS_DIR)
	delorean pipeline product-tests --test-config ./test-containers.yaml --output $(TEST_RESULTS_DIR) --namespace test-products

.PHONY: cluster/deploy
cluster/deploy: kustomize cluster/cleanup cluster/cleanup/crds cluster/prepare/crd cluster/prepare cluster/prepare/rbac/dedicated-admins deploy/integreatly-rhmi-cr.yml
	@ - oc create -f config/rbac/service_account.yaml
	@ - cd config/manager && $(KUSTOMIZE) edit set image controller=${IMAGE_FORMAT}
	@ - $(KUSTOMIZE) build config/redhat-$(INSTALLATION_SHORTHAND) | oc apply -f -

.PHONY: test/unit
test/unit: export WATCH_NAMESPACE=testing-namespaces-operator
test/unit:
	@TEMPLATE_PATH=$(TEMPLATE_PATH) ./scripts/ci/unit_test.sh

.PHONY: install/olm
install/olm: cluster/cleanup/olm cluster/cleanup/crds cluster/prepare cluster/prepare/olm/subscription deploy/integreatly-rhmi-cr.yml cluster/check/operator/deployment cluster/prepare/dms cluster/prepare/pagerduty

.PHONY: test/e2e/olm
test/e2e/olm: install/olm

.PHONY: cluster/deploy/integreatly-rhmi-cr.yml
cluster/deploy/integreatly-rhmi-cr.yml: deploy/integreatly-rhmi-cr.yml
	$(call wait_command, oc get RHMI $(INSTALLATION_NAME) -n $(NAMESPACE) --output=json -o jsonpath='{.status.stages.bootstrap.phase}' | grep -q completed, bootstrap phase, 5m, 30)
	$(call wait_command, oc get RHMI $(INSTALLATION_NAME) -n $(NAMESPACE) --output=json -o jsonpath='{.status.stages.installation.phase}' | grep -q completed, installation phase, 40m, 30)
ifeq ($(INSTALLATION_TYPE), managed)
	$(call wait_command, oc get RHMI $(INSTALLATION_NAME) -n $(NAMESPACE) --output=json -o jsonpath='{.status.stages.solution-explorer.phase}' | grep -q completed, solution-explorer phase, 10m, 30)
endif

.PHONY: cluster/prepare
cluster/prepare: cluster/prepare/project cluster/prepare/configmaps cluster/prepare/smtp cluster/prepare/pagerduty cluster/prepare/delorean cluster/prepare/quota

.PHONY: cluster/prepare/bundle
cluster/prepare/bundle: cluster/prepare/project cluster/prepare/configmaps cluster/prepare/smtp cluster/prepare/dms cluster/prepare/pagerduty cluster/prepare/delorean

.PHONY: create/olm/bundle
create/olm/bundle:
	./scripts/bundle-rhmi-operators.sh

.PHONY: create/3scale/index
create/3scale/index:
	./scripts/create-3scale-index.sh

.PHONY: cluster/prepare/project
cluster/prepare/project:
	@ - oc new-project $(NAMESPACE)
	@oc label namespace $(NAMESPACE) monitoring-key=middleware openshift.io/cluster-monitoring="true" --overwrite
	@oc project $(NAMESPACE)

.PHONY: kustomize cluster/prepare/configmaps
cluster/prepare/configmaps: kustomize
	$(KUSTOMIZE) build config/configmap | oc apply -n $(NAMESPACE) -f -

.PHONY: cluster/prepare/croaws
cluster/prepare/croaws:
	@ - oc create -f config/croaws/cro-aws-config.yml -n $(NAMESPACE)

.PHONY: cluster/prepare/crd
cluster/prepare/crd: kustomize
	$(KUSTOMIZE) build config/crd | oc apply -f -
	$(KUSTOMIZE) build config/crd-sandbox | oc apply -f -

.PHONY: cluster/prepare/local
cluster/prepare/local: kustomize cluster/prepare/project cluster/prepare/crd cluster/prepare/smtp cluster/prepare/dms cluster/prepare/pagerduty cluster/prepare/quota cluster/prepare/delorean cluster/prepare/croaws cluster/prepare/rbac/dedicated-admins
	@ - oc create -f config/rbac/service_account.yaml -n $(NAMESPACE)
	@ - $(KUSTOMIZE) build config/rbac-$(INSTALLATION_SHORTHAND) | oc create -f -

.PHONY: cluster/prepare/olm/subscription
cluster/prepare/olm/subscription:
	oc process -p NAMESPACE=$(NAMESPACE) -f config/olm/operator-subscription-template.yml | oc create -f - -n $(NAMESPACE)
	$(call wait_command, oc get crd rhmis.integreatly.org, rhmis.integreatly.org crd, 1m, 10)

.PHONY: cluster/check/operator/deployment
cluster/check/operator/deployment:
	$(call wait_command, oc get deployments rhmi-operator -n $(NAMESPACE) --output=json -o jsonpath='{.status.availableReplicas}' | grep -q 1, rhmi-operator ,2m, 10)

.PHONY: cluster/prepare/smtp
cluster/prepare/smtp:
	@-oc create secret generic $(NAMESPACE_PREFIX)smtp -n $(NAMESPACE) \
		--from-literal=host= \
		--from-literal=username= \
		--from-literal=password= \
		--from-literal=port= \
		--from-literal=tls=

.PHONY: cluster/prepare/pagerduty
cluster/prepare/pagerduty:
	@-oc create secret generic $(NAMESPACE_PREFIX)pagerduty -n $(NAMESPACE) \
		--from-literal=serviceKey=test

.PHONY: cluster/prepare/dms
cluster/prepare/dms:
	@-oc create secret generic $(NAMESPACE_PREFIX)deadmanssnitch -n $(NAMESPACE) \
		--from-literal=url=https://dms.example.com

.PHONY: cluster/prepare/quota
cluster/prepare/quota:
	@-oc process -n $(NAMESPACE) QUOTA=$(DEV_QUOTA) USERNAME=$(SMTP_USER) HOST=$(SMTP_ADDRESS) PASSWORD=$(SMTP_PASS) PORT=$(SMTP_PORT) FROM=$(SMTP_FROM) -f config/secrets/custom-addon-secret.yaml | oc apply -f -

.PHONY: cluster/prepare/quota/trial
cluster/prepare/quota/trial:
	@-oc process -n $(NAMESPACE) -f config/secrets/quota-trial-secret.yaml | oc apply -f -

.PHONY: cluster/prepare/delorean
cluster/prepare/delorean: cluster/prepare/delorean/pullsecret

.PHONY: cluster/prepare/delorean/pullsecret
cluster/prepare/delorean/pullsecret:
ifneq ( ,$(findstring image_mirror_mapping,$(IMAGE_MAPPINGS)))
	$(MAKE) setup/service_account
	./scripts/setup-delorean-pullsecret.sh
	$(MAKE) cluster/cleanup/serviceaccount
endif

.PHONY:cluster/prepare/rbac/dedicated-admins
cluster/prepare/rbac/dedicated-admins:
	@-oc create -f config/rbac/dedicated_admins_rbac.yaml

.PHONY: cluster/cleanup
cluster/cleanup: kustomize
	@-oc delete rhmis $(INSTALLATION_NAME) -n $(NAMESPACE) --timeout=240s --wait
	@-oc delete namespace $(NAMESPACE) --timeout=60s --wait
	@-$(KUSTOMIZE) build config/rbac-$(INSTALLATION_SHORTHAND) | oc delete -f -
	

.PHONY: cluster/cleanup/serviceaccount
cluster/cleanup/serviceaccount: kustomize
	@-oc delete serviceaccount ${TEMP_SERVICEACCOUNT_NAME} -n ${NAMESPACE}
	@-$(KUSTOMIZE) build config/rbac-$(INSTALLATION_SHORTHAND) | oc delete -f -

.PHONY: cluster/cleanup/olm
cluster/cleanup/olm: cluster/cleanup
	$(call wait_command, oc get projects -l integreatly=true -o jsonpath='{.items}' | grep -q '\[\]', integreatly namespace cleanup, 4m, 10)
	@-oc delete catalogsourceconfig.operators.coreos.com/installed-rhmi-operator -n openshift-marketplace
	@-oc delete operatorsource.operators.coreos.com/rhmi-operators -n openshift-marketplace

.PHONY: cluster/cleanup/crds
cluster/cleanup/crds:
	@-oc delete crd applicationmonitorings.applicationmonitoring.integreatly.org
	@-oc delete crd blackboxtargets.applicationmonitoring.integreatly.org
	@-oc delete crd grafanadashboards.integreatly.org
	@-oc delete crd grafanadatasources.integreatly.org
	@-oc delete crd grafanas.integreatly.org
	@-oc delete crd observabilities.observability.redhat.com
	@-oc delete crd rhmis.integreatly.org
	@-oc delete crd webapps.integreatly.org
	@-oc delete crd rhmiconfigs.integreatly.org
	@-oc delete crd apimanagementtenants.integreatly.org

.PHONY:cluster/cleanup/rbac/dedicated-admins
cluster/cleanup/rbac/dedicated-admins:
	@-oc delete -f config/rbac/dedicated_admins_rbac.yaml

.PHONY: deploy/integreatly-rhmi-cr.yml
deploy/integreatly-rhmi-cr.yml:
	@echo "selfSignedCerts = $(SELF_SIGNED_CERTS)"
	sed "s/INSTALLATION_NAME/$(INSTALLATION_NAME)/g" config/samples/integreatly-rhmi-cr.yaml | \
	sed "s/INSTALLATION_TYPE/$(INSTALLATION_TYPE)/g" | \
	sed "s/INSTALLATION_PREFIX/$(INSTALLATION_PREFIX)/g" | \
	sed "s/INSTALLATION_SHORTHAND/$(INSTALLATION_SHORTHAND)/g" | \
	sed "s/SELF_SIGNED_CERTS/$(SELF_SIGNED_CERTS)/g" | \
	sed "s/OPERATORS_IN_PRODUCT_NAMESPACE/$(OPERATORS_IN_PRODUCT_NAMESPACE)/g" | \
	sed "s/USE_CLUSTER_STORAGE/$(USE_CLUSTER_STORAGE)/g" > config/samples/integreatly-rhmi-cr.yml
	# Workaround until in_prow annotation can be removed from prow
	yq e -i '.metadata.annotations.in_prow="IN_PROW"' config/samples/integreatly-rhmi-cr.yml

	$(SED_INLINE) "s/IN_PROW/'$(IN_PROW)'/g" config/samples/integreatly-rhmi-cr.yml
	@-oc create -f config/samples/integreatly-rhmi-cr.yml

.PHONY: prepare-patch-release
prepare-patch-release:
	$(CONTAINER_ENGINE) pull quay.io/integreatly/delorean-cli:master
	$(CONTAINER_ENGINE) run --rm -e KUBECONFIG=/kube.config -v "${HOME}/.kube/config":/kube.config:z -v "${HOME}/.delorean.yaml:/.delorean.yaml" quay.io/integreatly/delorean-cli:master delorean release openshift-ci-release --config /.delorean.yaml --olmType $(OLMTYPE) --version $(TAG)

.PHONY: release/prepare
release/prepare: kustomize
	@KUSTOMIZE_PATH=$(KUSTOMIZE) ./scripts/prepare-release.sh

.PHONY: push/csv
push/csv:
	operator-courier verify packagemanifests/$(PROJECT)
	-operator-courier push packagemanifests/$(PROJECT)/ $(REPO) $(APPLICATION_REPO) $(TAG) "$(AUTH_TOKEN)"

.PHONY: gen/push/csv
gen/push/csv: release/prepare push/csv

# Generate namespace names to be used in docs
.PHONY: gen/namespaces
gen/namespaces:
	echo '// Generated file. Do not edit' > namespaces.asciidoc
	oc get namespace | \
	grep redhat-rhmi | \
	awk -S '{print"- "$$1}' >> namespaces.asciidoc

.PHONY: vendor/check
vendor/check: vendor/fix
	git diff --exit-code vendor/
	git diff --exit-code go.sum

.PHONY: vendor/check/prow
vendor/check/prow:
	sh scripts/setup-private-git-access.sh
	make vendor/check

.PHONY: vendor/fix
vendor/fix:
	go mod tidy
	go mod vendor

.PHONY: manifest/prodsec
manifest/prodsec:
	@./scripts/prodsec-manifest-generator.sh ${TYPE_OF_MANIFEST}

.PHONY: kubebuilder/check
kubebuilder/check: code/gen
	git diff --exit-code config/crd/bases
	git diff --exit-code config/rbac/role.yaml

# Generate bundle manifests and metadata, then validate generated files.
.PHONY: bundle
bundle: manifests kustomize
	$(OPERATOR_SDK) generate kustomize manifests --interactive=false -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(OPERATOR_IMAGE)
	$(KUSTOMIZE) build config/manifests | $(OPERATOR_SDK) generate bundle $(BUNDLE_GEN_FLAGS)
	$(OPERATOR_SDK) bundle validate ./bundle

.PHONY: bundle-rhmi
bundle-rhmi: manifests kustomize
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(OPERATOR_IMAGE)
	$(KUSTOMIZE) build config/manifests-rhmi | $(OPERATOR_SDK) generate bundle $(BUNDLE_GEN_FLAGS) --output-dir ./bundles/integreatly-operator/$(TAG)

.PHONY: bundle-rhoam
bundle-rhoam: manifests kustomize
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(OPERATOR_IMAGE)
	$(KUSTOMIZE) build config/manifests-rhoam | $(OPERATOR_SDK) generate bundle $(BUNDLE_GEN_FLAGS) --output-dir ./bundles/managed-api-service/$(TAG) --kustomize-dir config/manifests-rhoam

.PHONY: packagemanifests
packagemanifests: manifests kustomize
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(OPERATOR_IMAGE)
	$(KUSTOMIZE) build config/manifests-$(OPERATOR-TYPE) | $(OPERATOR_SDK) generate packagemanifests --kustomize-dir=config/manifests-$(OPERATOR-TYPE) --output-dir packagemanifests/$(OPERATOR-NAME) --version $(TAG)

# Build the bundle image.
.PHONY: bundle-build
bundle-build:
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

# USAGE: make olm/bundle BUNDLE_TAG="quay.io/mstoklus/integreatly-index:1.15.2" VERSION=1.15.2 OLM_TYPE=managed-api-service will build a bundle from 1.15.2 bundles/managed-api-service directory.
.PHONY: olm/bundle 
olm/bundle:
	docker build -f bundles/$(OLM_TYPE)/bundle.Dockerfile -t $(BUNDLE_TAG) --build-arg version=$(VERSION) .

.PHONY: coverage
coverage:
	hack/codecov.sh

.PHONY: commits/check
commits/check:
	@./scripts/commits-check.sh

.PHONY: gosec/exclude
gosec/exclude:
	gosec -exclude=G104,G107,G404,G601 ./...

##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest

## Tool Versions
KUSTOMIZE_VERSION ?= v4.5.2
CONTROLLER_TOOLS_VERSION ?= v0.8.0

KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/kustomize/kustomize/v4@$(KUSTOMIZE_VERSION)

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest