import '@fontsource-variable/geist';
import '@fontsource-variable/geist-mono';
import './app.css';
import { mount } from 'svelte';
import App from './App.svelte';
import Kit from './lib/ui/Kit.svelte';

const app = mount(import.meta.env.DEV && location.hash === '#kit' ? Kit : App, { target: document.getElementById('app')! });

export default app;
