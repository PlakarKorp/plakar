import DefaultLayout from "./DefaultLayout";
import {Container, Stack} from "@mui/material";

function TwoColumnLayout({leftComponent, rightComponent, conf}) {
    return (
        <DefaultLayout conf={conf}>
            <Stack sx={{p: 2, minHeight: '92vh'}} direction={'row'}>
                <Stack sx={{mr: 1, backgroundColor: 'white', p: 2, borderRadius: 2, width: '70%'}}>
                    {leftComponent}
                </Stack>
                <Stack sx={{ml: 1, backgroundColor: 'white', p: 2, borderRadius: 2, width: '30%'}}>
                    {rightComponent}
                </Stack>
            </Stack>
        </DefaultLayout>
    )
};

export default TwoColumnLayout;
