# Project Description

API Tool to integrate on AWS/GCP and manage network core resources as VPC Routes, Security Groups, Firewall Rules. Analyse connectivity and connection between resources.

# Tech Approaches

- TDD (Tests before create code)
- Hexagonal Architecture
- Service layer between handlers and repositories

# Architecture

```
HTTP Handlers → Services (Use Cases) → Ports (Interfaces) → Domain Models
                                            ↓
                        Secondary Adapters (AWS/GCP/Storage)
```

Provider is always a query parameter — routes are unified (no /aws/ or /gcp/ prefix).

# Routes

/vpc/describe/{vpcID}?provider=aws|gcp&account=<alias>&region=<r>
 - GET - describe VPC (list subnets, routes, peerings, VPNs)

/vpc/insert
 - POST - { provider, account, region, vpcID }

/security-rules/describe?provider=aws|gcp&account=<alias>&region=<r>&groupID=<id>
 - GET - describe security group based on id

/security-rules/insert
 - POST - { provider, account, region, groupID }

/security-rules/remove?provider=aws|gcp&account=<alias>&region=<r>&groupID=<id>&ruleID=<id>
 - DELETE - delete rule

/analyse
 - POST - multicloud connectivity analysis
 - { source_provider, source_account, source_region, source_vpc, dest_provider, dest_account, dest_region, destination_cidr }

/map?providers=aws,gcp&account=<alias>&region=<r>
 - GET - Overview map with VPCs and connections (peerings, VPNs) as graph

# Database

Files stored in infra/databases/text
