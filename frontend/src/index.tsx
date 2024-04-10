import React from 'react';
import ReactDOM from 'react-dom/client';
import 'bootstrap/dist/css/bootstrap.min.css';
import './index.css';
import reportWebVitals from './reportWebVitals';

import {QueryClient, QueryClientProvider} from "@tanstack/react-query";
import {Configuration, PrintersApi} from "./api";
import {PrintersAPIContext, PrintersAPIUrlBase} from "./app/printersAPIContext";
import App from "./app/App";


const queryClient = new QueryClient();

const basePath: string = process.env.NODE_ENV == "development" ?
    'http://localhost:8080/api/v1' : '/api/v1';
const printersAPI = new PrintersApi(new Configuration({basePath}), basePath);


const root = ReactDOM.createRoot(
    document.getElementById('root') as HTMLElement
);
root.render(
    <React.StrictMode>
        <QueryClientProvider client={queryClient}>
            <PrintersAPIContext.Provider value={printersAPI}>
                <PrintersAPIUrlBase.Provider value={basePath}>
                    <App/>
                </PrintersAPIUrlBase.Provider>
            </PrintersAPIContext.Provider>
        </QueryClientProvider>
    </React.StrictMode>
);

// If you want to start measuring performance in your app, pass a function
// to log results (for example: reportWebVitals(console.log))
// or send to an analytics endpoint. Learn more: https://bit.ly/CRA-vitals
reportWebVitals();
