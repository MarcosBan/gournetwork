# Project Description

API Tool to integrate on AWS/GCP and manage network core resources as VPC Routes, Security Groups, Firewall Rules. Analyse connectivity and connection between resources.

# Tech Approachs 

- TDD (Tests before create code)
- Hexagonal Architecture

# Routes 

/aws/vpc/describe/{vpc-id}
 - GET - describe (list subnetes and routes)

/aws/vpc/insert
 - POST
 - vpc-id
 - region
 - account

/gcp/vpc/describe/{vpc-id}
 - GET - describe (list subnets and routes)

/gcp/vpc/insert
 - POST
 - vpc-id
 - region
 - project-id

 /aws/security-rules/describe
  - GET - describe security group based on id rules

/aws/security-rules/insert
  - POST
  - securit-group-ud

/aws/security-rules/remove
  - DELETE - delete rule
  
/gcp/security-rules/describe
  - GET - describe security group based on id rules

/gcp/security-rules/remove
  - DELETE - delete rule

/gcp/security-rules/insert
 - POST - update rule

/analyse
 - POST - source vpc and destination vpc ip range (return if theres connection )

/map 
 - GET - Overview map connection in a structure master json

 # Database 

 Files stored in infra/databases/text