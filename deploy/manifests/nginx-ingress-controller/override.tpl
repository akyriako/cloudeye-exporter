controller:
  replicaCount: 1
  service:
    externalTrafficPolicy: Cluster
    annotations:
      kubernetes.io/elb.id: "${ELB_ID}"
