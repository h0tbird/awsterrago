generate:
	cp -R ~/git/hashicorp/terraform/dag/. pkg/dag
	cp -R ~/git/hashicorp/terraform/tfdiags/. pkg/tfd
	ls pkg/dag/* pkg/tfd/*
	gsed -i 's/tfdiags/tfd/g' pkg/dag/* pkg/tfd/*
	gsed -i 's_github.com/hashicorp/terraform/tfd_github.com/h0tbird/awsterrago/pkg/tfd_g' pkg/dag/*
	gsed -i '/github.com\/hashicorp\/terraform\/internal\/logging/d' pkg/dag/dag_test.go
	gofmt -w pkg
