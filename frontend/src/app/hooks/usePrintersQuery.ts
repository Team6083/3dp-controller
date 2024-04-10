import {useContext} from 'react';

import {PrintersAPIContext} from "@/app/printersAPIContext";
import {useQuery} from "@tanstack/react-query";


export function usePrintersQuery(refetchInterval: number = 2500) {
    const api = useContext(PrintersAPIContext);

    return useQuery({
        queryKey: ['printers'],
        queryFn: async () => {
            return api?.printersGet();
        },
        refetchInterval: refetchInterval,
        enabled: !!api,
    });
}
