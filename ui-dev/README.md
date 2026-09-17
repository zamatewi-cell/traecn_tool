# UI Development Workspace

This directory is dedicated to UI page development for the TraeCN Tools project.

## ? Structure

```
ui-dev/
©À©¤©¤ README.md              # This file
©À©¤©¤ package.json           # Node.js dependencies
©À©¤©¤ tsconfig.json          # TypeScript configuration
©À©¤©¤ vite.config.ts         # Vite build configuration
©À©¤©¤ tailwind.config.js     # Tailwind CSS configuration
©À©¤©¤ index.html             # Main HTML template
©À©¤©¤ src/
©¦   ©À©¤©¤ components/        # React components
©¦   ©À©¤©¤ pages/            # Page components
©¦   ©À©¤©¤ styles/           # Global styles
©¦   ©À©¤©¤ utils/            # Utility functions
©¦   ©À©¤©¤ hooks/            # Custom React hooks
©¦   ©À©¤©¤ types/            # TypeScript types
©¦   ©¸©¤©¤ assets/           # Static assets
©À©¤©¤ electron/
©¦   ©À©¤©¤ main.js           # Electron main process
©¦   ©¸©¤©¤ preload.js        # Electron preload script
©¸©¤©¤ release/              # Build outputs
```

## ? Purpose

This workspace is used by the UI development agent to:
- Design and implement React + TypeScript components
- Build Electron desktop application UI
- Create responsive layouts with Tailwind CSS
- Manage state with Zustand/Redux
- Implement OAuth login flows
- Build dashboard and management interfaces

## ? Quick Start

```bash
# Install dependencies
npm install

# Start development server
npm run dev

# Build for production
npm run build

# Build Electron app
npm run electron:dev
```

## ? Tech Stack

- **Frontend Framework**: React 18 + TypeScript
- **Build Tool**: Vite
- **Styling**: Tailwind CSS
- **Desktop**: Electron
- **State Management**: Zustand
- **HTTP Client**: Axios
- **Icons**: Lucide React

## ? Development Guidelines

1. **Component Structure**: Use functional components with hooks
2. **Type Safety**: All components must have TypeScript types
3. **Styling**: Use Tailwind CSS utility classes
4. **State**: Use Zustand for global state
5. **Testing**: Write unit tests for critical components

## ? Active Tasks

Track UI development tasks in the main docs folder:
- See: [`../docs/agents/ui-agent/README.md`](../docs/agents/ui-agent/README.md)

## ? Agent Workspace

This directory is managed by the **UI-Agent**, responsible for:
- UI/UX design implementation
- Component library development
- Responsive layout creation
- Theme and styling management
- Electron integration
