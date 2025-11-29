// Journey Map Manager - Google Maps integration for journey content visualization

let map;
let markers = [];
let infoWindows = [];
let activeInfoWindow = null;

// Initialize the map when Google Maps API loads
function initJourneyMap() {
    if (typeof google === 'undefined' || !google.maps) {
        console.error('Google Maps API not loaded');
        showMapError();
        return;
    }

    const mapElement = document.getElementById('map');
    if (!mapElement) {
        console.error('Map element not found');
        return;
    }

    // Check if we have content data
    if (!contentData || contentData.length === 0) {
        initEmptyMap(mapElement);
        return;
    }

    initMapWithContent(mapElement);
}

// Initialize empty map (no content with locations yet)
function initEmptyMap(mapElement) {
    map = new google.maps.Map(mapElement, {
        center: { lat: 0, lng: 0 },
        zoom: 2,
        mapTypeControl: true,
        streetViewControl: false,
        fullscreenControl: true,
    });

    // Add centered message overlay
    const overlay = document.createElement('div');
    overlay.style.cssText = `
        position: absolute;
        top: 50%;
        left: 50%;
        transform: translate(-50%, -50%);
        background: white;
        padding: 2rem;
        border-radius: 8px;
        box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        text-align: center;
        pointer-events: none;
    `;
    overlay.innerHTML = `
        <p style="color: #6c757d; margin: 0;">
            📍 Content with location data will appear here
        </p>
    `;
    mapElement.appendChild(overlay);
}

// Initialize map with content markers
function initMapWithContent(mapElement) {
    // Calculate bounds to fit all markers
    const bounds = new google.maps.LatLngBounds();
    contentData.forEach(content => {
        bounds.extend({ lat: content.lat, lng: content.lng });
    });

    // Create map centered on content
    const center = bounds.getCenter();
    map = new google.maps.Map(mapElement, {
        center: center,
        zoom: 10,
        mapTypeControl: true,
        streetViewControl: false,
        fullscreenControl: true,
        gestureHandling: 'greedy',
    });

    // Fit bounds to show all markers
    map.fitBounds(bounds);

    // Ensure minimum zoom level
    const listener = google.maps.event.addListener(map, 'idle', function() {
        if (map.getZoom() > 15) map.setZoom(15);
        google.maps.event.removeListener(listener);
    });

    // Create markers for each content item
    createMarkers();

    // Wire up content list interactions
    setupContentListeners();
}

// Create markers for all content with locations
function createMarkers() {
    contentData.forEach((content, index) => {
        const marker = new google.maps.Marker({
            position: { lat: content.lat, lng: content.lng },
            map: map,
            title: content.title,
            label: {
                text: String(index + 1),
                color: 'white',
                fontSize: '12px',
                fontWeight: 'bold'
            },
            animation: google.maps.Animation.DROP,
        });

        // Create info window
        const infoWindow = new google.maps.InfoWindow({
            content: createInfoWindowContent(content)
        });

        // Marker click opens info window and highlights content
        marker.addListener('click', () => {
            // Close previously open info window
            if (activeInfoWindow) {
                activeInfoWindow.close();
            }

            infoWindow.open(map, marker);
            activeInfoWindow = infoWindow;

            // Highlight and scroll to content item
            highlightContentItem(content.id);
            scrollToContentItem(content.id);
        });

        markers.push(marker);
        infoWindows.push(infoWindow);
    });
}

// Create HTML content for info window
function createInfoWindowContent(content) {
    const typeIcon = content.type === 'photo' ? '📷' : '🎥';
    return `
        <div style="max-width: 200px; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;">
            <h3 style="margin: 0 0 0.5rem; font-size: 0.875rem; color: #212529;">
                ${typeIcon} ${escapeHtml(content.title)}
            </h3>
            <p style="margin: 0; font-size: 0.75rem; color: #6c757d;">
                ${content.date}
            </p>
            <a href="/app/content/${content.id}" 
               style="display: inline-block; margin-top: 0.5rem; padding: 0.25rem 0.75rem; background: #007bff; color: white; text-decoration: none; border-radius: 4px; font-size: 0.75rem;"
               onclick="event.stopPropagation();">
                View Details
            </a>
        </div>
    `;
}

// Setup listeners for content list interactions
function setupContentListeners() {
    const contentItems = document.querySelectorAll('.content-item[data-lat][data-lng]');
    
    contentItems.forEach((item, index) => {
        // Hover shows marker bounce
        item.addEventListener('mouseenter', () => {
            if (markers[index]) {
                markers[index].setAnimation(google.maps.Animation.BOUNCE);
                setTimeout(() => {
                    if (markers[index]) {
                        markers[index].setAnimation(null);
                    }
                }, 700);
            }
        });

        // Click pans to marker and opens info window
        item.addEventListener('click', (e) => {
            // Only trigger if not clicking the link itself
            if (e.target.tagName !== 'A') {
                e.preventDefault();
                
                if (markers[index]) {
                    map.panTo(markers[index].getPosition());
                    map.setZoom(Math.max(map.getZoom(), 12));
                    
                    // Close previous info window
                    if (activeInfoWindow) {
                        activeInfoWindow.close();
                    }
                    
                    // Open this marker's info window
                    infoWindows[index].open(map, markers[index]);
                    activeInfoWindow = infoWindows[index];
                    
                    // Highlight content item
                    highlightContentItem(item.dataset.contentId);
                }
            }
        });
    });
}

// Highlight a content item in the list
function highlightContentItem(contentId) {
    // Remove previous highlights
    document.querySelectorAll('.content-item.highlighted').forEach(item => {
        item.classList.remove('highlighted');
    });

    // Add highlight to target
    const targetItem = document.querySelector(`.content-item[data-content-id="${contentId}"]`);
    if (targetItem) {
        targetItem.classList.add('highlighted');
    }
}

// Scroll content item into view
function scrollToContentItem(contentId) {
    const targetItem = document.querySelector(`.content-item[data-content-id="${contentId}"]`);
    if (targetItem) {
        targetItem.scrollIntoView({
            behavior: 'smooth',
            block: 'center'
        });
    }
}

// Escape HTML to prevent XSS
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// Show error message when map fails to load
function showMapError() {
    const mapElement = document.getElementById('map');
    if (mapElement) {
        mapElement.innerHTML = `
            <div style="display: flex; align-items: center; justify-content: center; height: 100%; padding: 2rem; text-align: center;">
                <div>
                    <p style="color: #dc3545; margin-bottom: 1rem;">⚠️ Map could not be loaded</p>
                    <p style="color: #6c757d; font-size: 0.875rem;">
                        Please check your API key configuration or internet connection.
                    </p>
                </div>
            </div>
        `;
    }
}

// Expose init function globally for Google Maps callback
window.initJourneyMap = initJourneyMap;
