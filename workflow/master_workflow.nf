#!/usr/bin/env nextflow

include {RUN_PREPROCESSOR} from './preprocessor.nf'
include {main_flow} from "${params.userJob}"
include {RUN_POSTPROCESSOR} from './postprocessor.nf'


workflow {
    RUN_PREPROCESSOR() | main_flow | RUN_POSTPROCESSOR | view { it.trim() }
}
