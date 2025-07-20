<execution>
  <constraint>
    ## Technical Constraints
    - **Go Language Limitation**: Must work within Go ecosystem and eino framework capabilities
    - **Local Processing Requirement**: All personal data processing must happen locally for privacy
    - **Resource Constraints**: System must run efficiently on personal hardware (not server-grade)
    - **Real-time Requirements**: Conversation responses must be sub-second for natural interaction
    - **Integration Limitations**: External API rate limits and service availability constraints
  </constraint>
  
  <rule>
    ## Mandatory Development Rules
    - **Privacy First**: Never send personal conversation data to external services without explicit consent
    - **Incremental Development**: Each component must be independently testable and deployable
    - **Error Handling**: All external integrations must have robust error handling and fallback mechanisms
    - **Logging Standards**: Comprehensive logging for debugging 24/7 operation issues
    - **Configuration Management**: All system parameters must be configurable without code changes
    - **Data Backup**: Implement automatic backup mechanisms for critical personal data
  </rule>
  
  <guideline>
    ## Development Best Practices
    - **Code Organization**: Follow Go project structure conventions with clear package separation
    - **Testing Strategy**: Unit tests for components, integration tests for data flow, end-to-end tests for user scenarios
    - **Documentation**: Maintain clear documentation for system architecture and API specifications
    - **Performance Monitoring**: Implement metrics collection for system performance analysis
    - **Security Practices**: Regular security audits and dependency vulnerability scanning
    - **User Feedback**: Build mechanisms to collect and incorporate user feedback for system improvement
  </guideline>
  
  <process>
    ## Development Workflow
    
    ### Step 1: Current State Analysis
    ```mermaid
    flowchart TD
        A[Analyze Existing Code] --> B[Identify Components]
        B --> C[Map Data Flow]
        C --> D[Document Architecture]
        D --> E[Identify Gaps]
    ```
    
    **Actions:**
    - Review existing eino integration code
    - Analyze neo4j schema and queries
    - Examine vector database implementation
    - Document current system capabilities and limitations
    
    ### Step 2: Architecture Design
    ```mermaid
    graph TD
        A[System Requirements] --> B[Component Design]
        B --> C[Data Model Design]
        C --> D[API Specification]
        D --> E[Integration Points]
        E --> F[Architecture Review]
    ```
    
    **Deliverables:**
    - System architecture diagram
    - Component interaction specifications
    - Data model schemas (Neo4j + Vector DB)
    - API documentation
    - Integration testing plan
    
    ### Step 3: Implementation Strategy
    ```mermaid
    flowchart LR
        A[Data Ingestion] --> B[Processing Pipeline]
        B --> C[Storage Layer]
        C --> D[Conversation Engine]
        D --> E[User Interface]
    ```
    
    **Implementation Order:**
    1. **Data Ingestion Service**: IM platform connectors and web conversation scrapers
    2. **Processing Pipeline**: Content filtering, context extraction, memory consolidation
    3. **Storage Integration**: Enhanced Neo4j and vector database operations
    4. **Conversation Engine**: Proactive triggers and dialogue management
    5. **User Interface**: TTS integration and conversation interface
    
    ### Step 4: Testing and Validation
    ```mermaid
    graph TD
        A[Unit Testing] --> B[Integration Testing]
        B --> C[Performance Testing]
        C --> D[User Acceptance Testing]
        D --> E[Security Testing]
    ```
    
    **Testing Approach:**
    - Component-level unit tests with mock dependencies
    - Integration tests for data flow between components
    - Performance tests for 24/7 operation scenarios
    - User experience testing for conversation quality
    - Security testing for data protection
    
    ### Step 5: Deployment and Monitoring
    ```mermaid
    flowchart TD
        A[Local Deployment] --> B[Process Management]
        B --> C[Health Monitoring]
        C --> D[Performance Metrics]
        D --> E[User Feedback]
        E --> F[Continuous Improvement]
    ```
    
    **Deployment Strategy:**
    - Containerized deployment for easy management
    - Systemd service configuration for auto-restart
    - Log aggregation and monitoring setup
    - Performance metrics dashboard
    - User feedback collection mechanism
  </process>
  
  <criteria>
    ## Success Criteria
    
    ### Functional Requirements
    - ✅ System runs continuously for 24/7 without manual intervention
    - ✅ Successfully ingests data from configured IM platforms
    - ✅ Maintains conversation context across sessions
    - ✅ Proactively initiates relevant conversations
    - ✅ Responds to user queries within 1 second
    - ✅ Supports both voice and text interactions
    
    ### Performance Requirements
    - ✅ Memory usage remains stable over extended operation
    - ✅ CPU usage stays below 20% during normal operation
    - ✅ Storage growth is manageable with automatic cleanup
    - ✅ Response time for conversation queries < 500ms
    - ✅ System recovery time after failure < 30 seconds
    
    ### Quality Requirements
    - ✅ Conversation quality feels natural and contextually relevant
    - ✅ Privacy protection meets personal security standards
    - ✅ System reliability > 99% uptime
    - ✅ Data integrity maintained across all storage systems
    - ✅ User satisfaction with proactive conversation timing
    
    ### Technical Requirements
    - ✅ Code coverage > 80% for critical components
    - ✅ All external integrations have proper error handling
    - ✅ System configuration is externalized and hot-reloadable
    - ✅ Comprehensive logging for debugging and monitoring
    - ✅ Security vulnerabilities addressed and documented
  </criteria>
</execution>
