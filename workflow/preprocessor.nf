process RUN_PREPROCESSOR {
    container 'alpine'
    memory '1GB'
    output: stdout

    script:
    """
    #!/bin/sh
    echo '$params.workflowJobId'
    python /build_virtual_path.py /job/workflow/input.csv
    echo 'end preprocessing'
    """
}
