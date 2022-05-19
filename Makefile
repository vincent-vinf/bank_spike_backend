VERSION = 0.0.1
WORK_DIR = .
REGISTRY = registry.cn-qingdao.aliyuncs.com/adpc/

all_image: access_image spike_image user_image admin_image order_image

build_push: all_image push

access_image:
	docker build --target access -t $(REGISTRY)spike_access-service:$(VERSION) $(WORK_DIR)

spike_image:
	docker build --target spike -t $(REGISTRY)spike_spike-service:$(VERSION) $(WORK_DIR)

user_image:
	docker build --target user -t $(REGISTRY)spike_user-service:$(VERSION) $(WORK_DIR)

admin_image:
	docker build --target admin -t $(REGISTRY)spike_admin-service:$(VERSION) $(WORK_DIR)

order_image:
	docker build --target order -t $(REGISTRY)spike_order-service:$(VERSION) $(WORK_DIR)

push:
	docker push $(REGISTRY)spike_access-service:$(VERSION)
	docker push $(REGISTRY)spike_spike-service:$(VERSION)
	docker push $(REGISTRY)spike_user-service:$(VERSION)
	docker push $(REGISTRY)spike_admin-service:$(VERSION)
	docker push $(REGISTRY)spike_order-service:$(VERSION)

