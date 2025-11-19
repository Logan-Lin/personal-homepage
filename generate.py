import os
import yaml
from datetime import datetime
from jinja2 import Environment, FileSystemLoader


if __name__ == '__main__':
    with open('data.yaml', 'r') as file:
        profile_data = yaml.safe_load(file)

    env = Environment(loader=FileSystemLoader('templates'))
    current_year = datetime.now().year

    os.makedirs('dist', exist_ok=True)
    os.makedirs('dist/publications', exist_ok=True)
    os.makedirs('dist/projects', exist_ok=True)
    os.makedirs('dist/presentations', exist_ok=True)
    os.makedirs('dist/teaching', exist_ok=True)

    # Create .nojekyll file to prevent GitHub Pages from using Jekyll
    with open('dist/.nojekyll', 'w') as f:
        pass

    def render_template(template_name, output_path, **kwargs):
        template = env.get_template(template_name)
        html = template.render(year=current_year, **kwargs)

        with open(output_path, 'w') as file:
            file.write(html)

        print(f'Generated {output_path}')

    # Always generate the main index page
    render_template('index.html', 'dist/index.html', data=profile_data, is_home_page=True)
    
    # Generate individual section pages only if they're in the sections list
    sections = profile_data.get('sections', [])
    
    if 'publications' in sections:
        render_template('publications.html', 'dist/publications/index.html', data=profile_data, is_home_page=False)
    
    if 'projects' in sections:
        render_template('projects.html', 'dist/projects/index.html', data=profile_data, is_home_page=False)
    
    if 'presentations' in sections:
        render_template('presentations.html', 'dist/presentations/index.html', data=profile_data, is_home_page=False)
    
    if 'teaching' in sections:
        render_template('teaching.html', 'dist/teaching/index.html', data=profile_data, is_home_page=False)
    
    print('Static site generation complete!')
