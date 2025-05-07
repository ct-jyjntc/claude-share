# Frontend Dockerfile
FROM node:20-alpine

# Set working directory
WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm install

# Copy the rest of the application
COPY . .

# Build the application
RUN npm run build

# Expose port for the web server
EXPOSE 5173
EXPOSE 3001

# Start the application
CMD ["npm", "run", "start"]
