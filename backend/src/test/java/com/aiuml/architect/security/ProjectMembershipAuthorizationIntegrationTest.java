package com.aiuml.architect.security;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.AutoConfigureMockMvc;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.http.MediaType;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.transaction.annotation.Transactional;
import org.springframework.test.web.servlet.MockMvc;

import java.util.UUID;

import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;

/** Exercises the HTTP bearer filter, controller, and membership service against PostgreSQL. */
@SpringBootTest
@AutoConfigureMockMvc
@Transactional
class ProjectMembershipAuthorizationIntegrationTest {
  @Autowired MockMvc mockMvc;
  @Autowired ObjectMapper mapper;
  @Autowired JdbcTemplate jdbc;

  @Test
  void assignedProjectsReturnPersistedDiagramCount() throws Exception {
    String login = mockMvc.perform(post("/api/v1/auth/login").contentType(MediaType.APPLICATION_JSON)
            .content("{\"email\":\"ana@example.com\",\"password\":\"Password123!\"}"))
        .andExpect(status().isOk()).andReturn().getResponse().getContentAsString();
    String token = mapper.readTree(login).path("accessToken").asText();

    mockMvc.perform(get("/api/v1/projects").header("Authorization", "Bearer " + token))
        .andExpect(status().isOk())
        .andExpect(jsonPath("$[?(@.id == 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa')].diagramCount").value(1));
  }

  @Test
  void memberCannotReadAnotherExistingProjectDiagrams() throws Exception {
    UUID projectId = UUID.randomUUID();
    // Bruno owns this real project. Ana is deliberately not added as a member.
    jdbc.update("INSERT INTO projects(id,name,description,access_code,owner_id) VALUES (?,?,?,?,?)",
        projectId, "Private architecture", "Authorization test", "TEST" + projectId.toString().substring(0, 6),
        UUID.fromString("22222222-2222-2222-2222-222222222222"));
    jdbc.update("INSERT INTO project_memberships(project_id,user_id,role) VALUES (?,?,?)",
        projectId, UUID.fromString("22222222-2222-2222-2222-222222222222"), "OWNER");

    String login = mockMvc.perform(post("/api/v1/auth/login").contentType(MediaType.APPLICATION_JSON)
            .content("{\"email\":\"ana@example.com\",\"password\":\"Password123!\"}"))
        .andExpect(status().isOk()).andReturn().getResponse().getContentAsString();
    String token = mapper.readTree(login).path("accessToken").asText();

    mockMvc.perform(get("/api/v1/projects/{projectId}/diagrams", projectId)
            .header("Authorization", "Bearer " + token))
        .andExpect(status().isForbidden());
  }
}
