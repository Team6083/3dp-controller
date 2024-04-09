import {createContext} from "react";

import {PrintersApi} from "@/api";

export const PrintersAPIContext = createContext<PrintersApi | undefined>(undefined);

