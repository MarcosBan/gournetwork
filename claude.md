# Project Description

API Tool to integrate on AWS/GCP and manage network core resources as VPC Routes, Security Groups, Firewall Rules. Analyse connectivity and connection between resources.

# Tech Approachs 

- TDD (Tests before create code)
- Hexagonal Architecture

# Routes 

/aws/vpc/
 - GET - describe (list subnetes and routes)
 - POST - update (routes)
/gcp/vpc/
 - GET - describe (list subnets and routes)
 - POST - update (routes)

 /aws/security-rules/describe
  - GET - describe security group based on id rules
  - POST - update rule
  - DELETE - delete rule
  
/gcp/security-rules/describe
  - GET - describe security group based on id rules
  - POST - update rule
  - DELETE - delete rule

/analyse
 - POST - source vpc and destination vpc ip range (return if theres connection )

/map 
 - GET - Overview map connection in a structure master json