import React from "react";
import ProjectCard, { ProjectCardProps } from "./ProjectCard";
import AddProjectCard from "./AddProjectCard.tsx";

interface ProjectGridProps {
  envId: string;
  envName: string;
  projects: ProjectCardProps[];
  onProjectDeleted?: () => void;
}

const ProjectGrid: React.FC<ProjectGridProps> = ({ envId, envName, projects, onProjectDeleted }) => {
  return (
  <>
    <div className="flex justify-between items-end mb-12">
      <div>
        <h2 className="text-4xl font-bold mt-6 font-headline tracking-tight text-on-surface">
          {envName}
        </h2>
        <p className="text-on-surface-variant font-body mt-2">
          <span className="text-secondary font-semibold">
            {projects.length} project{projects.length !== 1 ? "s" : ""}
          </span>{" "}
          in this environment.
        </p>
      </div>
    </div>

    <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-4">
      <AddProjectCard envId={envId} />
      {projects.map((project) => (
        <ProjectCard key={project.id} {...project} envId={envId} onDeleted={onProjectDeleted} />
      ))}
    </div>
  </>
  )
}

export default ProjectGrid;
