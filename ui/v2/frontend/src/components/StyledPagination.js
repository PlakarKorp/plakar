import {Pagination, Stack, Typography} from "@mui/material";
import {fetchSnapshots} from "../utils/PlakarApiClient";
import React from "react";

const empty = () => {}

const StyledPagination = ({pageCount=0, showFirstButton=true, showLastButton=true, onChange=empty}) => {
    return (
        <Stack sx={{width: "100%"}} alignItems="stretch" direction={'row'}
               justifyContent={"flex-start"} padding={2}>
            <Pagination count={pageCount}
                        color={'primary'}
                        size={'small'}
                        showFirstButton={showFirstButton}
                        showLastButton={showLastButton}
                        onChange={(event, page) => {
                            onChange(event, page)
                        }}
            />

            {/*<TablePagination*/}

            {/*    showFirstButton*/}
            {/*    showLastButton*/}
            {/*    rowsPerPageOptions={[5, 10, 25, {label: 'All', value: -1}]}*/}
            {/*    colSpan={3}*/}
            {/*    count={rows.length}*/}
            {/*    rowsPerPage={10}*/}
            {/*    page={page}*/}
            {/*    SelectProps={{*/}
            {/*        inputProps: {*/}
            {/*            'aria-label': 'rows per page',*/}
            {/*        },*/}
            {/*        native: true,*/}
            {/*    }}*/}

            {/*/>*/}
        </Stack>)
}

export default StyledPagination;