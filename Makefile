VERSION = 0.0.2
WORK_DIR = .
REGISTRY = registry.cn-qingdao.aliyuncs.com/adpc/

all_image: access_image spike_image user_image admin_image order_image

build_push: all_image push

access_image:
	docker build --target access -t $(REGISTRY)spike-access-service:$(VERSION) $(WORK_DIR)

spike_image:
	docker build --target spike -t $(REGISTRY)spike-spike-service:$(VERSION) $(WORK_DIR)

user_image:
	docker build --target user -t $(REGISTRY)spike-user-service:$(VERSION) $(WORK_DIR)

admin_image:
	docker build --target admin -t $(REGISTRY)spike-admin-service:$(VERSION) $(WORK_DIR)

order_image:
	docker build --target order -t $(REGISTRY)spike-order-service:$(VERSION) $(WORK_DIR)

push:
	docker push $(REGISTRY)spike-access-service:$(VERSION)
	docker push $(REGISTRY)spike-spike-service:$(VERSION)
	docker push $(REGISTRY)spike-user-service:$(VERSION)
	docker push $(REGISTRY)spike-admin-service:$(VERSION)
	docker push $(REGISTRY)spike-order-service:$(VERSION)

