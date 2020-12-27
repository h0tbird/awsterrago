# awsterrago

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

Terraform DAG
```
cp -r ~/git/hashicorp/terraform/{dag,tfdiags} pkg
mv pkg/tfdiags pkg/tfd
sed -i 's/tfdiags/tfd/g' pkg/dag/* pkg/tfd/*
sed -i 's_github.com/hashicorp/terraform/tfd_github.com/h0tbird/awsterrago/pkg/tfd_g' pkg/dag/*
```
