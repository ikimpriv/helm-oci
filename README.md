helm-oci
================
Helm-oci is a command-line tool that helps in listing and deleting helm charts/tags in OCI repositories.  
This tool has only been tested with a standard Docker registry.  
Helm-oci uses helm or Docker configuration files and/or Docker credential tools to retrieve the repository credentials.

Examples:
-----------------

- To list Helm charts:

      helm-oci ls oci://registry.example.com/repository   

- To list available tags of a chart:

      helm-oci tags oci://registry.example.com/repository/mychart   

- To delete tag 0.1.1 of a chart named mychart (note that deletion must be enabled in the repository):

      helm-oci rm oci://registry.example.com/repository/mychart 0.1.1