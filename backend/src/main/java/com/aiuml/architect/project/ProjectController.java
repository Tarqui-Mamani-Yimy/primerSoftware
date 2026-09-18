package com.aiuml.architect.project;
import org.springframework.security.core.Authentication; import org.springframework.web.bind.annotation.*; import java.util.*;
@RestController @RequestMapping("/api/v1/projects") class ProjectController {private final ProjectService service; ProjectController(ProjectService s){service=s;} @GetMapping ProjectService.ProjectResponse[] assigned(Authentication a){return service.assigned(UUID.fromString(a.getName())).toArray(ProjectService.ProjectResponse[]::new);} }
