ucconfig
========

ucconfig (UserClouds Config) enables declarative configuration of UserClouds
resources.

ucconfig is implemented as a usability layer on top of Terraform. It is not as
powerful or flexible as Terraform, but it avoids many of the common
frustrations of working with Terraform, such as manually writing configuration
or managing Terraform state. Since ucconfig generates Terraform configuration
and state, you can "eject" from ucconfig and manage UserClouds resources
directly with Terraform at any point.
