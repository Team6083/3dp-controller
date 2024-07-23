import React from 'react'
import ReactDOM from 'react-dom/client'
import 'bootstrap/dist/css/bootstrap.min.css'
import './index.css';
import {QueryClient, QueryClientProvider} from "@tanstack/react-query"

import {Configuration, PrintersApi} from "./api"
import App from './app/App.tsx'
import {PrintersAPIContext, PrintersAPIUrlBase} from "./app/printersAPIContext"


const queryClient = new QueryClient();

const basePath = import.meta.env.VITE_API_URL;
const printersAPI = new PrintersApi(new Configuration({basePath}), basePath);


ReactDOM.createRoot(document.getElementById('root')!).render(
    <React.StrictMode>
        <QueryClientProvider client={queryClient}>
            <PrintersAPIContext.Provider value={printersAPI}>
                <PrintersAPIUrlBase.Provider value={basePath}>
                    <App/>
                </PrintersAPIUrlBase.Provider>
            </PrintersAPIContext.Provider>
        </QueryClientProvider>
    </React.StrictMode>,
)
