#!/bin/bash

# Function to check if required tools are installed
check_requirements() {
    if ! command -v fswatch &> /dev/null && ! command -v inotifywait &> /dev/null; then
        echo "Error: Neither fswatch nor inotify-tools is installed"
        echo "Please install one of them:"
        echo "For macOS: brew install fswatch"
        echo "For Linux: sudo apt-get install inotify-tools"
        exit 1
    fi
}

# Function to kill existing SAM process
kill_sam() {
    # Kill any process using port 3000
    local port_pid=$(lsof -ti:3000)
    if [ ! -z "$port_pid" ]; then
        echo "Killing process on port 3000..."
        kill -9 $port_pid 2>/dev/null || true
    fi

    # Kill existing SAM process if we have its PID
    if [ ! -z "$sam_pid" ]; then
        echo "Stopping existing SAM process..."
        kill $sam_pid 2>/dev/null || true
        wait $sam_pid 2>/dev/null || true
    fi

    # Small delay to ensure port is freed
    sleep 1
}

# Build and start function
build_and_start() {
    # Kill existing SAM process if it exists
    kill_sam
    
    # Run the original build script
    build_function() {
        local service=$1
        echo "Building $service..."
        
        cd src/$service
        
        echo "Building binary for $service..."
        CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
            -ldflags="-s -w" \
            -o bootstrap \
            main.go
        
        chmod 755 bootstrap
        
        echo "Creating zip for $service..."
        # Remove existing zip if it exists
        rm -f function.zip
        
        # Create new zip file
        zip function.zip bootstrap > /dev/null 2>&1
        
        rm -f bootstrap
        
        cd ../..
        echo "Successfully built $service"
    }

    # Build each function
    for service in "auth_service" "user_service" "workspace_service" "project_service" "folder_service" "item_service"; do
        build_function "$service"
    done

    # Start SAM in background
    echo "Starting SAM..."
    sam local start-api -t template.local.yaml --skip-pull-image &
    sam_pid=$!
}

# Variables for debouncing
last_build_time=0
debounce_delay=2  # seconds

# Main watch function
watch_and_rebuild() {
    # Updated watched directories to include cmd, internal, and pkg
    local watched_dirs="src cmd internal pkg"
    
    # Initial build
    build_and_start
    
    echo "Watching for changes in $watched_dirs..."
    
    if command -v fswatch &> /dev/null; then
        # macOS/fswatch version
        fswatch -o \
            --exclude '\.zip$' \
            --exclude 'bootstrap$' \
            --exclude '\.git' \
            --exclude '\.DS_Store' \
            --event Updated \
            --event Created \
            -e ".*" \
            -i "\.go$" \
            $watched_dirs | while read f; do
            current_time=$(date +%s)
            time_since_last=$((current_time - last_build_time))
            
            if [ $time_since_last -ge $debounce_delay ]; then
                echo "Change detected in Go files"
                build_and_start
                last_build_time=$current_time
            fi
        done
    else
        # Linux/inotifywait version
        while true; do
            inotifywait -r \
                -e modify,create \
                --exclude '(\.zip$|bootstrap$|\.git|\.DS_Store)' \
                --format '%w%f' \
                -q \
                --include '\.go$' \
                $watched_dirs
            
            current_time=$(date +%s)
            time_since_last=$((current_time - last_build_time))
            
            if [ $time_since_last -ge $debounce_delay ]; then
                echo "Change detected in Go files"
                build_and_start
                last_build_time=$current_time
            fi
        done
    fi
}

# Cleanup function
cleanup() {
    echo "Cleaning up..."
    kill_sam
    exit 0
}

# Set up trap for cleanup
trap cleanup SIGINT SIGTERM

# Main execution
check_requirements
watch_and_rebuild