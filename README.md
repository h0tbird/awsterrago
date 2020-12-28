# terramorph

```
aws iam list-policies | jq '.Policies[] | select(.PolicyName | contains("cluster-api-provider-aws"))'
aws iam list-roles | jq '.Roles[] | select(.RoleName | contains("cluster-api-provider-aws"))'
aws iam list-instance-profiles | jq '.InstanceProfiles[] | select(.InstanceProfileName | contains("cluster-api-provider-aws"))'
```

```
aws iam list-attached-role-policies --role-name control-plane.cluster-api-provider-aws.sigs.k8s.io
aws iam list-attached-role-policies --role-name controllers.cluster-api-provider-aws.sigs.k8s.io
aws iam list-attached-role-policies --role-name nodes.cluster-api-provider-aws.sigs.k8s.io
```
