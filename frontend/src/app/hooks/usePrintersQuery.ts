import {useContext} from 'react';

import {PrintersAPIContext} from "@/app/printersAPIContext";
import {useQuery} from "@tanstack/react-query";


export function usePrintersQuery() {
    const api = useContext(PrintersAPIContext);

    return useQuery({
        queryKey: ['printers'],
        queryFn: async () => {
            return api?.printersGet();
        },
        refetchInterval: 5000,
        enabled: !!api,
    });
}
