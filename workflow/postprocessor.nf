process RUN_POSTPROCESSOR {
    container 'alpine'
    memory '1GB'
    input:
    val postprocessor_input

    output:stdout

    script:
    """
    #! /bin/sh
    echo 'hello postprocessor $postprocessor_input'
    """
}
